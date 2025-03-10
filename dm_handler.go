package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	httpClientTimeoutDuration        = time.Second * 5
	transportIdleConnTimeoutDuration = time.Second * 90
	baseBskyURL                      = "https://bsky.social/xrpc"
)

var (
	errUnauthorized = fmt.Errorf("unauthorized")
	errExpiredToken = fmt.Errorf("expired token")
)

type auth struct {
	AccessJwt  string `json:"accessJwt"`
	RefershJWT string `json:"refreshJwt"`
	Did        string `json:"did"`
}

type accessData struct {
	handle      string
	appPassword string
}

type ListConvosResponse struct {
	Cursor string  `json:"cursor"`
	Convos []Convo `json:"convos"`
}

type Convo struct {
	ID          string        `json:"id"`
	Members     []ConvoMember `json:"members"`
	UnreadCount int           `json:"unreadCount"`
}

type ConvoMember struct {
	Did    string `json:"did"`
	Handle string `json:"handle"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResp struct {
	Messages []Message `json:"messages"`
	Cursor   string    `json:"cursor"`
}

type Message struct {
	ID     string        `json:"id"`
	Sender MessageSender `json:"sender"`
	Text   string        `json:"text"`
	Embed  MessageEmbed  `json:"embed"`
}

type MessageEmbed struct {
	Record MessageEmbedRecord `json:"record"`
}

type MessageEmbedRecord struct {
	URI    string                   `json:"uri"`
	Author MessageEmbedRecordAuthor `json:"author"`
	Value  MessageEmbedPost         `json:"value"`
}

type MessageEmbedRecordAuthor struct {
	Did    string `json:"did"`
	Handle string `json:"handle"`
}

type MessageEmbedPost struct {
	Text string `json:"text"`
}

type MessageSender struct {
	Did string `json:"did"`
}

type UpdateMessageReadRequest struct {
	ConvoID   string `json:"convoId"`
	MessageID string `json:"messageId"`
}

type DmService struct {
	httpClient    *http.Client
	accessData    accessData
	auth          auth
	timerDuration time.Duration
	pdsURL        string
	bookmarkStore BookmarkStore
}

func NewDmService(bookmarkStore BookmarkStore, timerDuration time.Duration) (*DmService, error) {
	httpClient := http.Client{
		Timeout: httpClientTimeoutDuration,
		Transport: &http.Transport{
			IdleConnTimeout: transportIdleConnTimeoutDuration,
		},
	}

	accessHandle := os.Getenv("MESSAGING_ACCESS_HANDLE")
	accessAppPassword := os.Getenv("MESSAGING_ACCESS_APP_PASSWORD")
	pdsURL := os.Getenv("MESSAGING_PDS_URL")

	service := DmService{
		httpClient: &httpClient,
		accessData: accessData{
			handle:      accessHandle,
			appPassword: accessAppPassword,
		},
		timerDuration: timerDuration,
		pdsURL:        pdsURL,
		bookmarkStore: bookmarkStore,
	}

	auth, err := service.Authenicate()
	if err != nil {
		return nil, fmt.Errorf("authenticating: %w", err)
	}

	service.auth = auth

	return &service, nil
}

func (d *DmService) Start(ctx context.Context) {
	timer := time.NewTimer(d.timerDuration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Warn("context canceled - stopping dm task")
			return
		case <-timer.C:
			err := d.HandleMessageTimer(ctx)
			if err != nil {
				slog.Error("handle message timer", "error", err)
			}
			timer.Reset(d.timerDuration)
		}
	}
}

func (d *DmService) HandleMessageTimer(ctx context.Context) error {
	convoResp, err := d.GetUnreadMessages()
	if err != nil {
		return fmt.Errorf("get unread messages: %w", err)
	}

	// TODO: handle the cursor pagination

	for _, convo := range convoResp.Convos {
		if convo.UnreadCount == 0 {
			continue
		}

		messageResp, err := d.GetMessages(ctx, convo.ID)
		if err != nil {
			slog.Error("failed to get messages for convo", "error", err, "convo id", convo.ID)
			continue
		}

		unreadCount := convo.UnreadCount
		unreadMessages := make([]Message, 0, convo.UnreadCount)
		// TODO: handle cursor pagination
		for _, msg := range messageResp.Messages {
			// TODO: techincally if I get to a message that's from the bot account, then there shouldn't be
			// an more unread messages?
			if msg.Sender.Did == d.auth.Did {
				continue
			}

			unreadMessages = append(unreadMessages, msg)
			unreadCount--
			if unreadCount == 0 {
				break
			}
		}

		for _, msg := range unreadMessages {
			d.handleMessage(msg)

			err = d.MarkMessageRead(msg.ID, convo.ID)
			if err != nil {
				slog.Error("marking message read", "error", err)
				continue
			}
		}
	}

	return nil
}

func (d *DmService) handleMessage(msg Message) {
	// for now, ignore messages that don't have linked posts in them
	if msg.Embed.Record.URI == "" {
		return
	}

	rkey := getRKeyFromATURI(msg.Embed.Record.URI)
	msgAction := strings.ToLower(msg.Text)

	var err error

	switch {
	case strings.Contains(msgAction, "delete"):
		err = d.handleDeleteBookmark(msg)
	default:
		err = d.handleCreateBookmark(msg)
	}

	if err != nil {
		// TODO: perhaps continue here so that we don't mark the message as read so it can be tried again? Or perhaps send a message
		// too the user?
		slog.Error("failed to handle bookmark message", "error", err, "rkey", rkey, "sender", msg.Sender.Did)
	}
}

func (d *DmService) handleCreateBookmark(msg Message) error {
	content := msg.Embed.Record.Value.Text
	if len(content) > 75 {
		content = fmt.Sprintf("%s...", content[:75])
	}

	publicURI := getPublicPostURIFromATURI(msg.Embed.Record.URI, msg.Embed.Record.Author.Handle)

	rkey := getRKeyFromATURI(msg.Embed.Record.URI)

	err := d.bookmarkStore.CreateBookmark(rkey, publicURI, msg.Embed.Record.URI, msg.Embed.Record.Author.Did, msg.Embed.Record.Author.Handle, msg.Sender.Did, content)
	if err != nil {
		return fmt.Errorf("creating bookmark: %w", err)
	}
	return nil
}

func (d *DmService) handleDeleteBookmark(msg Message) error {
	rkey := getRKeyFromATURI(msg.Embed.Record.URI)

	err := d.bookmarkStore.DeleteFeedPostsForBookmarkedPostURIandUserDID(msg.Embed.Record.URI, msg.Sender.Did)
	if err != nil {
		return fmt.Errorf("failed to delete feed posts of replies to bookmark for user: %w", err)
	}

	err = d.bookmarkStore.DeleteBookmark(rkey, msg.Sender.Did)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	return nil
}

func getPublicPostURIFromATURI(atURI, authorHandle string) string {
	atSplit := strings.Split(atURI, "app.bsky.feed.post")
	if len(atSplit) < 2 {
		slog.Error("can't get public post URI from AT uri", "at uri", atURI)
		return ""
	}
	return fmt.Sprintf("https://bsky.app/profile/%s/post%s", authorHandle, atSplit[1])
}

func (d *DmService) GetUnreadMessages() (ListConvosResponse, error) {
	url := fmt.Sprintf("%s/xrpc/chat.bsky.convo.listConvos?readState=unread", d.pdsURL)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ListConvosResponse{}, fmt.Errorf("create new list convos http request: %w", err)
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Atproto-Proxy", "did:web:api.bsky.chat#bsky_chat")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", d.auth.AccessJwt))

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return ListConvosResponse{}, fmt.Errorf("do http request to list convos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		err = decodeResp(resp.Body, &errorResp)
		if err != nil {
			return ListConvosResponse{}, err
		}

		return ListConvosResponse{}, fmt.Errorf("listing convos responded with code %d: %s", resp.StatusCode, errorResp.Error)
	}

	var listConvoResp ListConvosResponse
	err = decodeResp(resp.Body, &listConvoResp)
	if err != nil {
		return ListConvosResponse{}, err
	}

	return listConvoResp, nil
}

func (d *DmService) MarkMessageRead(messageID, convoID string) error {
	bodyReq := UpdateMessageReadRequest{
		ConvoID:   convoID,
		MessageID: messageID,
	}

	bodyB, err := json.Marshal(bodyReq)
	if err != nil {
		return fmt.Errorf("marshal update message request body: %w", err)
	}

	r := bytes.NewReader(bodyB)

	url := fmt.Sprintf("%s/xrpc/chat.bsky.convo.updateRead", d.pdsURL)
	request, err := http.NewRequest("POST", url, r)
	if err != nil {
		return fmt.Errorf("create new list convos http request: %w", err)
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Atproto-Proxy", "did:web:api.bsky.chat#bsky_chat")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", d.auth.AccessJwt))

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("do http request to update message read: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var errorResp ErrorResponse
	err = decodeResp(resp.Body, &errorResp)
	if err != nil {
		return err
	}

	return fmt.Errorf("listing convos responded with code %d: %s", resp.StatusCode, errorResp.Error)

}

func (d *DmService) Authenicate() (auth, error) {
	url := fmt.Sprintf("%s/com.atproto.server.createSession", baseBskyURL)

	requestData := map[string]interface{}{
		"identifier": d.accessData.handle,
		"password":   d.accessData.appPassword,
	}

	data, err := json.Marshal(requestData)
	if err != nil {
		return auth{}, errors.Wrap(err, "failed to marshal request")
	}

	r := bytes.NewReader(data)

	request, err := http.NewRequest("POST", url, r)
	if err != nil {
		return auth{}, errors.Wrap(err, "failed to create request")
	}

	request.Header.Add("Content-Type", "application/json")

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return auth{}, errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	var loginResp auth
	err = decodeResp(resp.Body, &loginResp)
	if err != nil {
		return auth{}, err
	}

	return loginResp, nil
}

func (d *DmService) RefreshTask(ctx context.Context) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			err := d.RefreshAuthenication(ctx)
			if err != nil {
				slog.Error("handle refresh auth timer", "error", err)
				// TODO: better retry with backoff probably
				timer.Reset(time.Minute)
				continue
			}
			timer.Reset(time.Hour)
		}
	}
}

func (d *DmService) RefreshAuthenication(ctx context.Context) error {
	url := fmt.Sprintf("%s/com.atproto.server.refreshSession", baseBskyURL)

	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", d.auth.RefershJWT))

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to make request")
	}
	defer resp.Body.Close()

	var loginResp auth
	err = decodeResp(resp.Body, &loginResp)
	if err != nil {
		return err
	}

	d.auth = loginResp

	return nil
}

func (d *DmService) GetMessages(ctx context.Context, convoID string) (MessageResp, error) {
	url := fmt.Sprintf("%s/xrpc/chat.bsky.convo.getMessages?convoId=%s", d.pdsURL, convoID)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return MessageResp{}, fmt.Errorf("create new get messages http request: %w", err)
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Atproto-Proxy", "did:web:api.bsky.chat#bsky_chat")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", d.auth.AccessJwt))

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return MessageResp{}, fmt.Errorf("do http request to get messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		err = decodeResp(resp.Body, &errorResp)
		if err != nil {
			return MessageResp{}, err
		}

		return MessageResp{}, fmt.Errorf("listing convos responded with code %d: %s", resp.StatusCode, errorResp.Error)
	}

	var messageResp MessageResp
	err = decodeResp(resp.Body, &messageResp)
	if err != nil {
		return MessageResp{}, err
	}

	return messageResp, nil
}

func decodeResp(body io.Reader, result any) error {
	resBody, err := io.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "failed to read response")
	}

	err = json.Unmarshal(resBody, result)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal response")
	}
	return nil
}

css:
	tailwindcss -i app.css -o public/styles.css --watch

templ:
	templ generate --watch --proxy="http://localhost:8090" --open-browser=false -v
air:
	air
dev:
	make -j3 templ css air

docker:
	@docker build -f Dockerfile -t willdot/templ-demo .
	@docker push willdot/templ-demo

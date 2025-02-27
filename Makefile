default: up

build: 
	@mkdir -p bin
	@sudo env GOOS=linux GOARCH=arm64 GOFLAGS=-buildvcs=false go build -o ./bin

clean: down
	@sudo rm -rf ./bin

image: build
	@sudo docker build --no-cache -t otimofie/tennis_bot:latest .

status:
	@sudo docker-compose ps

up: image
	# @sudo docker-compose up --build -d
	@sudo docker-compose up --build

down:
	@sudo docker-compose down -v
	@sudo docker image prune --filter="dangling=true" -f

docker_update:
	@sudo rm -rf ~/Library/Caches/com.docker.docker ~/Library/Cookies/com.docker.docker.binarycookies ~/Library/Group\ Containers/group.com.docker ~/Library/Logs/Docker\ Desktop ~/Library/Preferences/com.docker.docker.plist ~/Library/Preferences/com.electron.docker-frontend.plist ~/Library/Saved\ Application\ State/com.electron.docker-frontend.savedState ~/.docker /Library/LaunchDaemons/com.docker.vmnetd.plist /Library/PrivilegedHelperTools/com.docker.vmnetd /usr/local/lib/docker

docker_prepare:
	@rm ~/.docker/config.json

static_check: build
	@sudo ./scripts/linters.sh
# TODO: enter to psql
# TODO: backup and restore db

PHONY: build clean image status up down docker_update docker_prepare static_check

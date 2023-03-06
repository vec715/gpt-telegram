deploy:
	gcloud run deploy openai --region europe-central2 --source .

build:
	go build -o bin/chatgpt-telegram cmd/main.go

docker-compose:
	docker compose up -d

podman-compose:
	podman-compose up -d
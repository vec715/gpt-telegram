# ChatGTP Telegram

The project involves integrating ChatGPT, which is a large language model, into the Telegram messaging platform. This integration would allow for private conversations with ChatGPT, where users could ask the model questions or engage in conversation with it. Additionally, users would also create topic-related groups where they could discuss specific subjects with ChatGPT's assistance.

Currently, the project supports the features to run project locally via Docker compose and on Google Cloud Platform. The project is still in development and more features will be added in the future.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Docker](#docker)
- [License](#license)

## Installation

### Prerequisites

- Go (>=1.19)
- A Telegram bot with a token
- OpenAI API token
- Docker or Podman

### Build from source

1. Clone the repository
2. Create an `.env` file in the project root directory. Use `.env.example.redis` (to use Redis as storage. Preferred choice) or `.env.example.gcp` (to use Google Datastore as persistent storage. Requires complex setup) as a template.
3. Fill in the environment variables in the `.env` file with the appropriate values.

Run the following command to build the binary:

```shell
make build
```

And then run the following command to run the binary:

```shell
./bin/chatgpt-telegram
```

## Docker

To build the Docker image, run the following command:

```shell
make docker-compose
```

## Podman

To build the Podman image, run the following command:

```shell
make podman-compose
```

## Usage

Just send a message to the bot and it will respond to you.
Or you can create a group and add the bot to it. Then, you can send messages to the group and the bot will respond to you. But dont forget to add the permission to the bot to read messages in the group.


The service listens for Telegram bot updates and generates responses using OpenAI's GPT-3 API.

## License

This project is licensed under the MIT License.

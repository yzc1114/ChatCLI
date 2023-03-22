if [ "$(id -u)" -ne 0 ]; then
        echo 'This script must be run by sudo privilege' >&2
        exit 1
fi

curl -L https://github.com/yzc1114/ChatCLI/releases/download/v0.0.1/chat-cli-linux-x86-v0.0.1 > chat-cli && mv chat-cli /usr/local/bin && chmod +x /usr/local/bin/chat-cli && echo "Please set the environment variable OPENAI_API_KEY to access ChatGPT"

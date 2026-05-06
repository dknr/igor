# igor

A chatbot that listens for @mentions on a grunt server and generates LLM-powered responses.

## Configuration

Create `~/.config/igor/config.yaml`:

```yaml
grunt:
  server_addr: "http://localhost:54765"
  user_id: "igor"
  password: "<your password>"
  mention: "@igor"
llm:
  base_url: "http://localhost:8080"
  model: "llama-3.2-3b-instruct"
  api_key: ""
igor:
  system_prompt: "You are igor, a simple LLM agent. Respond succinctly, in a gruff and terse manner."
```

## Usage

```bash
./igor
```

## License

ISC
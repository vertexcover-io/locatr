Create a .env file and add the following credentials.

```
SUPABASE_URL=
SUPABASE_KEY=
ANTHROPIC_API_KEY=
COHERE_API_KEY=
```

### Fetch evals from supabase
  ```
  uv run fetch_evals.py
  ```

### Generating results

- Original Locatr
  > Run `go mod init; go mod tidy` to install dependencies
  ```
  go run locatr.go
  ```

- Anthropic Grounding Locatr
  ```
  uv run grounding_locatr.py anthropic
  ```

- OS Atlas Grounding Locatr
  ```
  uv run grounding_locatr.py os_atlas
  ```
  
### Comparing results with evals

- Original Locatr
  ```
  uv run compare.py original
  ```

- Anthropic Grounding Locatr
  ```
  uv run compare.py anthropic
  ```

- OS Atlas Grounding Locatr
  ```
  uv run compare.py os_atlas
  ```

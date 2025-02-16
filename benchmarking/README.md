Create a .env file and add the following credentials.

```
SUPABASE_URL=
SUPABASE_KEY=
ANTHROPIC_API_KEY=
COHERE_API_KEY=
```

### Fetch evals from supabase
  ```
  python fetch_evals.py
  ```

### Generating results

- Original Locatr
  > Run `go mod init; go mod tidy` to install dependencies
  ```
  go run locatr.go
  ```

- Anthropic Grounding Locatr
  ```
  python grounding_locatr.py anthropic
  ```

- OS Atlas Grounding Locatr
  ```
  python grounding_locatr.py os_atlas
  ```
  
### Comparing results with evals

- Original Locatr
  ```
  python compare.py original
  ```

- Anthropic Grounding Locatr
  ```
  python compare.py anthropic
  ```

- OS Atlas Grounding Locatr
  ```
  python compare.py os_atlas
  ```

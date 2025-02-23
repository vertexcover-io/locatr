## Start the interactive evaluator

```sh
go run evaluator.go -input urls.txt
```
> This will write to a JsON lines file `results_{<start-time>}.jsonl`

## Conver the json lines output to CSV and Markdown

```sh
python convert.py results_{<start-time>}.jsonl
```

---

### About annotated screenshots

Each color in the annotated screenshots represents a different selection method:

- 🔴 Red: Manual user selection
- 🔵 Blue: Original locatr (with reranker)
- 🟢 Green: Anthropic grounding locatr
## Start the interactive evaluator

```sh
go run evaluator.go -input urls.txt
```
> This will write to a JsON lines file `results_{<start-time>}.jsonl`

## Render the final evaluation

```sh
python render.py results_{<start-time>}.jsonl
```
> This will write to a Markdown file `rendered_results_{<start-time>}.md`

import json
import sys
from pathlib import Path
from collections import defaultdict

def jsonl_to_markdown(input_file, output_file):
    """Convert JSONL file to Markdown format with URL grouping."""
    
    # First, group entries by URL
    url_entries = defaultdict(list)
    with open(input_file, 'r') as f_in:
        for line in f_in:
            entry = json.loads(line.strip())
            url_entries[entry['Url']].append(entry)
    
    with open(output_file, 'w') as f_out:
        for url, entries in url_entries.items():
            # Write URL as main heading
            f_out.write(f"# URL: {url}\n\n")
            
            # Process each entry for this URL
            for i, entry in enumerate(entries, 1):
                f_out.write(f"## Entry {i}\n\n")
                
                f_out.write(f"**Description**: {entry['ElementDescription']}\n\n")
                f_out.write(f"**Coordinates**: X={entry['ElementCoordinates']['X']}, Y={entry['ElementCoordinates']['Y']}\n\n")
                f_out.write(f"**Scroll To**: X={entry['ScrollCoordinates']['X']}, Y={entry['ScrollCoordinates']['Y']}\n\n")
                
                for model, details in entry['Outputs'].items():
                    f_out.write(f"#### `{model}`\n")
                    
                    # List all generated points
                    f_out.write("- Generated Points:\n")
                    for j, point in enumerate(details['GeneratedPoints'], 1):
                        f_out.write(f"  - Point {j}: X={point['X']}, Y={point['Y']}\n")
                    f_out.write("\n")
                    
                    f_out.write(f"- Input Tokens: {details['InputTokens']}\n")
                    f_out.write(f"- Output Tokens: {details['OutputTokens']}\n")
                    f_out.write(f"- Total Tokens: {details['TotalTokens']}\n\n")
                
                # Write image path if exists
                if 'ImagePath' in entry:
                    f_out.write(f"### Annotated Screenshot\n")
                    f_out.write(f"![Screenshot]({entry['ImagePath']})\n\n")
            
            # Add horizontal line between URLs
            f_out.write("---\n\n")

if __name__ == "__main__":
    input_file = Path(sys.argv[1]).with_suffix(".jsonl")
    output_file = Path(f"rendered_{input_file.stem}.md")
    
    # Create output directory if it doesn't exist
    Path(output_file).parent.mkdir(parents=True, exist_ok=True)
    
    jsonl_to_markdown(input_file, output_file)
    print(f"Conversion complete. Output written to {output_file}")
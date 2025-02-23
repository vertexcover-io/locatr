import json
import sys
import csv
from pathlib import Path
from collections import defaultdict


def format_points(points):
    """Format list of points into a semicolon-separated string."""
    if not points:
        return ""
    return ";".join(f"({p['X']:.2f},{p['Y']:.2f})" for p in points)


def jsonl_to_csv(input_file, output_file):
    """Convert JSONL file to CSV format."""
    # Define CSV headers
    headers = [
        "Url",
        "ScrollCoordinates_X",
        "ScrollCoordinates_Y",
        "ElementDescription",
        "ElementCoordinates_X",
        "ElementCoordinates_Y",
        "OriginalLocatr_InputTokens",
        "OriginalLocatr_OutputTokens",
        "OriginalLocatr_TotalTokens",
        "OriginalLocatr_CostInDollars",
        "OriginalLocatr_GeneratedPoints",
        "AnthropicGroundingLocatr_InputTokens",
        "AnthropicGroundingLocatr_OutputTokens",
        "AnthropicGroundingLocatr_TotalTokens",
        "AnthropicGroundingLocatr_CostInDollars",
        "AnthropicGroundingLocatr_GeneratedPoints",
        "ImagePath",
    ]

    with open(output_file, "w", newline="") as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=headers)
        writer.writeheader()

        with open(input_file, "r") as jsonl:
            for line in jsonl:
                entry = json.loads(line.strip())

                # Prepare the CSV row
                row = {
                    "Url": entry["Url"],
                    "ScrollCoordinates_X": entry["ScrollCoordinates"]["X"],
                    "ScrollCoordinates_Y": entry["ScrollCoordinates"]["Y"],
                    "ElementDescription": entry["ElementDescription"],
                    "ElementCoordinates_X": entry["ElementCoordinates"]["X"],
                    "ElementCoordinates_Y": entry["ElementCoordinates"]["Y"],
                    "ImagePath": entry.get("ImagePath", ""),
                }

                # Add data for each model
                for model in ["originalLocatr", "anthropicGroundingLocatr"]:
                    prefix = model.replace(
                        "originalLocatr", "OriginalLocatr"
                    ).replace(
                        "anthropicGroundingLocatr", "AnthropicGroundingLocatr"
                    )
                    if model in entry["Outputs"] and entry["Outputs"][model]:
                        output = entry["Outputs"][model]
                        row.update(
                            {
                                f"{prefix}_InputTokens": output["InputTokens"],
                                f"{prefix}_OutputTokens": output[
                                    "OutputTokens"
                                ],
                                f"{prefix}_TotalTokens": output["TotalTokens"],
                                f"{prefix}_CostInDollars": output[
                                    "CostInDollars"
                                ],
                                f"{prefix}_GeneratedPoints": format_points(
                                    output.get("GeneratedPoints", [])
                                ),
                            }
                        )
                    else:
                        # Fill with empty values if model data is missing
                        row.update(
                            {
                                f"{prefix}_InputTokens": "",
                                f"{prefix}_OutputTokens": "",
                                f"{prefix}_TotalTokens": "",
                                f"{prefix}_CostInDollars": "",
                                f"{prefix}_GeneratedPoints": "",
                            }
                        )

                writer.writerow(row)


def jsonl_to_markdown(input_file, output_file):
    """Convert JSONL file to Markdown format with URL grouping."""

    # First, group entries by URL
    url_entries = defaultdict(list)
    with open(input_file, "r") as f_in:
        for line in f_in:
            entry = json.loads(line.strip())
            url_entries[entry["Url"]].append(entry)

    with open(output_file, "w") as f_out:
        for url, entries in url_entries.items():
            # Write URL as main heading
            f_out.write(f"# URL: {url}\n\n")

            # Process each entry for this URL
            for i, entry in enumerate(entries, 1):
                f_out.write(f"## Entry {i}\n\n")

                f_out.write(
                    f"**Description**: {entry['ElementDescription']}\n\n"
                )
                f_out.write(
                    f"**Coordinates**: X={entry['ElementCoordinates']['X']}, Y={entry['ElementCoordinates']['Y']}\n\n"
                )
                f_out.write(
                    f"**Scroll To**: X={entry['ScrollCoordinates']['X']}, Y={entry['ScrollCoordinates']['Y']}\n\n"
                )

                for model, details in entry["Outputs"].items():
                    f_out.write(f"#### `{model}`\n")

                    # List all generated points
                    f_out.write("- Generated Points:\n")
                    for j, point in enumerate(details["GeneratedPoints"], 1):
                        f_out.write(
                            f"  - Point {j}: X={point['X']}, Y={point['Y']}\n"
                        )
                    f_out.write("\n")

                    f_out.write(f"- Input Tokens: {details['InputTokens']}\n")
                    f_out.write(f"- Output Tokens: {details['OutputTokens']}\n")
                    f_out.write(f"- Total Tokens: {details['TotalTokens']}\n")
                    f_out.write(
                        f"- Cost in Dollars: {details['CostInDollars']}\n\n"
                    )

                # Write image path if exists
                if "ImagePath" in entry:
                    f_out.write("### Annotated Screenshot\n")
                    f_out.write(f"![Screenshot]({entry['ImagePath']})\n\n")

            # Add horizontal line between URLs
            f_out.write("---\n\n")


if __name__ == "__main__":
    input_file = Path(sys.argv[1]).with_suffix(".jsonl")

    # Create both markdown and CSV outputs
    markdown_output = input_file.with_suffix(".md")
    csv_output = input_file.with_suffix(".csv")

    jsonl_to_markdown(input_file, markdown_output)
    jsonl_to_csv(input_file, csv_output)
    print("Conversion complete. Outputs written to:")
    print(f"- Markdown: {markdown_output}")
    print(f"- CSV: {csv_output}")

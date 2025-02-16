import os
import yaml
import re
from urllib.parse import urlparse, urlunparse
from supabase import create_client
from postgrest.exceptions import APIError

# Initialize Supabase client
supabase= create_client(os.environ["SUPABASE_URL"], os.environ["SUPABASE_KEY"])

def sanitize_filename(url: str) -> str:
    """Sanitizes the filename by replacing invalid characters."""
    parsed_url = urlparse(url)
    unparsed_url = urlunparse(["", parsed_url.netloc, parsed_url.path, "", "", ""])
    return re.sub(r'[^a-zA-Z0-9]', '_', unparsed_url.strip().strip("/")).lower()

def generate_yaml(table_name: str, directory: str):
    """Fetches data from Supabase and generates YAML files per unique URL."""
    
    # Ensure directory exists
    os.makedirs(directory, exist_ok=True)

    # Fetch data from Supabase
    try:
        response = supabase.table(table_name).select("*").execute()
    except APIError as e:
        print("Error fetching data:", e)
        return
    
    data = response.data

    # Group by page_url
    grouped_data = {}
    for row in data:
        url = row["page_url"]
        if url not in grouped_data:
            grouped_data[url] = []
        grouped_data[url].append({
            "userRequest": row["element_description"]["hard"],
            "locatrs": row["expected_selectors"]
        })

    # Write each unique URL's YAML to a file
    for url, steps in grouped_data.items():
        file_name = os.path.join(directory, f"{sanitize_filename(url)}.yaml")
        yaml_content = {"url": url, "steps": steps}

        with open(file_name, "w", encoding="utf-8") as file:
            yaml.dump(yaml_content, file, default_flow_style=False, allow_unicode=True)

        print(f"Generated: {file_name}")

# Example usage
generate_yaml("eval", "schema/eval")

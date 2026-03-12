import json

with open("config.json", "r") as f:
    config = json.load(f)

config["admin_pass_hash"] = ""

with open("config.json", "w") as f:
    json.dump(config, f, indent=2)

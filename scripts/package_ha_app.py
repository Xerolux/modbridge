import re
from pathlib import Path
from zipfile import ZIP_DEFLATED, ZipFile

root = Path(__file__).resolve().parents[1]
app = root / "ha-app"
version = (root / "version.txt").read_text(encoding="utf-8").strip()
output = root / "bin" / f"modbridge-ha-{version}.zip"
output.parent.mkdir(exist_ok=True)

with ZipFile(output, "w", ZIP_DEFLATED) as archive:
    def add_file(path):
        archive.write(root / path, f"modbridge/{path.as_posix()}")

    for name in (".dockerignore", "go.mod", "go.sum", "version.txt", "LICENSE",
                 "frontend/index.html", "frontend/package.json", "frontend/package-lock.json",
                 "frontend/vite.config.js", "frontend/postcss.config.js", "frontend/tailwind.config.js"):
        add_file(Path(name))
    for path in sorted(root.glob("*.go")):
        add_file(path.relative_to(root))
    for directory, pattern in (("pkg", "*.go"), ("frontend/src", "*"), ("frontend/public", "*")):
        for path in sorted((root / directory).rglob(pattern)):
            if path.is_file():
                add_file(path.relative_to(root))

    dockerfile = (root / "Dockerfile").read_text(encoding="utf-8")
    archive.writestr("modbridge/Dockerfile", dockerfile + "\n" + (app / "Dockerfile.local").read_text(encoding="utf-8"))
    config = (app / "config.yaml").read_text(encoding="utf-8")
    archive.writestr("modbridge/config.yaml", re.sub(r"(?m)^version: .*", f'version: "{version}"', config))
    archive.writestr("modbridge/run.sh", (app / "run.sh").read_text(encoding="utf-8"))
    archive.write(app / "DOCS.md", "modbridge/DOCS.md")

print(output)

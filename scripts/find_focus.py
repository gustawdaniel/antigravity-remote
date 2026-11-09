with open("/opt/antigravity-ide/resources/app/out/jetskiAgent/main.js", "rb") as f:
    content = f.read()

import re
matches = [m.start() for m in re.finditer(b"LanguageServerService", content)]
for pos in matches[:10]:
    print("MATCH AT", pos)
    print(content[max(0, pos-50):min(len(content), pos+150)].decode("utf-8", errors="replace"))

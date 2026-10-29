"""Render this project's deliberately small Markdown subset as offline, printable RTL HTML."""
from html import escape
from pathlib import Path
import re
root=Path(__file__).resolve().parents[1]
css='''*{box-sizing:border-box}body{margin:0;background:#f3f6fb;color:#213049;font:16px/1.8 Arial,"Segoe UI",sans-serif}main{max-width:870px;margin:36px auto;background:white;padding:42px 54px;border-top:6px solid #4666e9;border-radius:8px;box-shadow:0 7px 30px #1724380a}header{font:bold 20px Arial;color:#4666e9;direction:ltr;text-align:right;letter-spacing:1px}h1{font-size:28px;margin:18px 0 12px}h2{font-size:20px;margin:30px 0 10px;color:#31498e;break-after:avoid}p,li{font-size:15px}li{margin:7px 0}code{direction:ltr;unicode-bidi:embed;display:inline-block;font:13px/1.7 Consolas,monospace;background:#eef2fa;border-radius:4px;padding:1px 5px;overflow-wrap:anywhere}footer{border-top:1px solid #dfe5ee;padding-top:18px;margin-top:35px;color:#7d899c;font-size:12px}a{color:#4666e9}@media(max-width:650px){main{margin:0;padding:25px 20px;border-radius:0}h1{font-size:24px}}@media print{body{background:#fff}main{margin:0;max-width:none;padding:0;border:0;box-shadow:none}h2{margin-top:22px}p,li{font-size:11pt}footer{font-size:9pt}a{color:inherit}li{break-inside:avoid}@page{size:A4;margin:18mm}}'''
def inline(s):
 s=escape(s)
 s=re.sub(r'`([^`]+)`',r'<code>\1</code>',s)
 return re.sub(r'\*\*([^*]+)\*\*',r'<strong>\1</strong>',s)
def render(text):
 out=[];list_type=None
 def close_list():
  nonlocal list_type
  if list_type:out.append(f'</{list_type}>');list_type=None
 for line in text.splitlines():
  if not line.strip():close_list();continue
  if line.startswith('#'):
   close_list();level=len(line)-len(line.lstrip('#'));out.append(f'<h{level}>{inline(line[level:].strip())}</h{level}>');continue
  match=re.match(r'^(?:\d+\. |(-) )(.*)$',line)
  if match:
   kind='ul' if match[1] else 'ol'
   if kind!=list_type:
    close_list();list_type=kind;out.append(f'<{kind}>')
   out.append('<li>'+inline(match[2])+'</li>');continue
  close_list();out.append('<p>'+inline(line)+'</p>')
 close_list();return '\n'.join(out)
for name in ['USER_GUIDE_HE','INSTALL_SUPPORT_HE']:
 src=root/'packaging'/f'{name}.md';text=src.read_text();title=text.splitlines()[0].lstrip('# ')
 result=f'<!doctype html>\n<html lang="he" dir="rtl"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{escape(title)}</title><style>{css}</style></head><body><main><header>Nimbus Phonebook</header>{render(text)}<footer>Nimbus Phonebook · גרסה 1.1 · אוקטובר 2026</footer></main></body></html>\n'
 (root/'packaging'/f'{name}.html').write_text(result)
 print('Rendered',name)

'use strict';
const $ = id => document.getElementById(id);
const token = document.querySelector('meta[name="phonebook-token"]').content;
let saved = { revision: 0, contacts: [], updatedAt: null }, contacts = [], info = null, dirty = false, busy = false, ready = false, editID = null, preview = null;
const formatDate = value => !value || value.startsWith('0001-') ? 'טרם נשמר' : new Intl.DateTimeFormat('he-IL', { day:'2-digit', month:'2-digit', year:'numeric', hour:'2-digit', minute:'2-digit' }).format(new Date(value));
function element(tag, cls, value) { const el=document.createElement(tag); if(cls)el.className=cls; if(value!==undefined)el.textContent=value; return el; }
function notice(message, error=false) { $('notice').textContent=message; $('notice').classList.remove('hidden'); $('notice').classList.toggle('error',error); }
async function api(path, method='GET', body) {
 const options={method,headers:{}};
 if(body!==undefined){ options.headers['Content-Type']='application/json'; options.headers['X-Phonebook-Token']=token; options.body=JSON.stringify(body); }
 const response=await fetch(path,options); const data=await response.json();
 if(!response.ok){ const e=new Error(data.errors ? data.errors.join('\n') : data.error || 'הבקשה נכשלה'); e.status=response.status; throw e; }
 return data;
}
function controls() { $('save').disabled=!ready||!dirty||busy; $('reload').disabled=busy; for(const id of ['add-contact','empty-add','import-csv'])$(id).disabled=!ready||busy; document.body.classList.toggle('dirty',dirty); $('save-title').textContent=dirty?'יש שינויים שטרם נשמרו':'כל השינויים שמורים'; $('save-description').textContent=dirty?'שמרו את השינויים כדי לעדכן את הספר הזמין לטלפונים.':'הטלפונים יקבלו את הספר המעודכן ברענון הבא שלהם.'; $('save-indicator').textContent=dirty?'•':'✓'; }
function setDirty(){dirty=true; $('notice').classList.add('hidden'); render();}
function render() {
 const term=$('search').value.trim().toLocaleLowerCase('he');
 const matches=contacts.filter(c=>[c.name,...c.phones].some(v=>v.toLocaleLowerCase('he').includes(term)));
 $('total-count').textContent=contacts.length.toLocaleString('he-IL'); $('list-count').textContent=contacts.length; $('nav-count').textContent=contacts.length;
 $('shown-count').textContent=`${matches.length} אנשי קשר${term?' מתוך '+contacts.length:''}`;
 $('results-count').textContent=term?`${matches.length} תוצאות חיפוש`:'כל אנשי הקשר';
 $('empty').classList.toggle('hidden',contacts.length!==0); $('no-results').classList.toggle('hidden',contacts.length===0||matches.length!==0);
 $('contacts-body').replaceChildren();
 for(const [i,c] of matches.entries()){
  const tr=element('tr');tr.append(element('td','row-number',i+1));
  const td=element('td'); const person=element('div','person-cell');person.append(element('span','person-avatar',c.name.trim().split(/\s+/).slice(0,2).map(n=>Array.from(n)[0]).join('')),element('span','',c.name));td.append(person);tr.append(td);
  const primary=element('td');primary.append(element('span','phone',c.phones[0]));tr.append(primary);
  const extra=element('td');const numbers=element('div','additional');for(const p of c.phones.slice(1))numbers.append(element('span','phone',p));if(!c.phones.slice(1).some(Boolean))numbers.append(element('span','','—'));extra.append(numbers);tr.append(extra);
  const actions=element('td');const row=element('div','row-actions');
  const edit=element('button','icon-button','✎');edit.setAttribute('aria-label',`עריכת ${c.name}`);edit.title='עריכה';edit.disabled=busy;edit.onclick=()=>openEditor(c);
  const del=element('button','icon-button delete-button','×');del.setAttribute('aria-label',`מחיקת ${c.name}`);del.title='מחיקה';del.disabled=busy;del.onclick=async()=>{if(await confirmAction('מחיקת איש קשר',`למחוק את ${c.name} מהספר? המחיקה תיכנס לתוקף לאחר שמירת השינויים.`,'אישור מחיקה')){contacts=contacts.filter(v=>v.id!==c.id);setDirty();}};
  row.append(edit,del);actions.append(row);tr.append(actions);$('contacts-body').append(tr);
 }
 $('last-saved').textContent=formatDate(saved.updatedAt);$('revision').textContent=`גרסה ${saved.revision} · ${saved.revision?'נשמרה במחשב המקומי':'אין שינויים שמורים'}`; controls();
}
async function load(){busy=true;controls();try{const [b,i]=await Promise.all([api('/api/book'),api('/api/info')]);saved=b;contacts=structuredClone(b.contacts);info=i;dirty=false;ready=true;$('max-contacts').textContent=i.limits.contacts.toLocaleString('he-IL');$('degraded').classList.toggle('hidden',!i.degraded);$('contact-name').maxLength=i.limits.name;for(const id of ['phone1','phone2','phone3'])$(id).maxLength=i.limits.number;$('connection').classList.remove('offline');$('connection').replaceChildren(element('i'),document.createTextNode(' המערכת פעילה'));}catch(e){ready=false;notice('לא ניתן לטעון את הספר: '+e.message,true);$('connection').textContent='החיבור נכשל';$('connection').classList.add('offline');}finally{busy=false;render();}}
function openEditor(c){editID=c?.id||null;$('contact-form').reset();$('editor-error').textContent='';$('editor-title').textContent=c?'עריכת איש קשר':'איש קשר חדש';$('editor-submit').textContent=c?'עדכון איש קשר':'הוספה לספר';if(c){$('contact-name').value=c.name;c.phones.forEach((p,i)=>$('phone'+(i+1)).value=p);}$('editor').showModal();$('contact-name').focus();}
$('contact-form').addEventListener('submit',event=>{event.preventDefault();const name=$('contact-name').value.trim();let phones=['phone1','phone2','phone3'].map(id=>$(id).value.trim());while(phones.length>1&&!phones.at(-1))phones.pop();if(!name||!phones[0]||phones.some(p=>!/^[0-9+*#,; ()-]*$/.test(p))){$('editor-error').textContent='הזינו שם ומספר תקין. מותרות ספרות והסימנים + * # , ; - ( ).';return;}if(!editID&&contacts.length>=info.limits.contacts){$('editor-error').textContent='הספר הגיע למגבלת אנשי הקשר.';return;}const duplicate=contacts.some(c=>c.id!==editID&&(c.name===name||c.phones.some(p=>p&&phones.includes(p))));const c={id:editID||crypto.randomUUID(),name,phones};if(editID)contacts=contacts.map(v=>v.id===editID?c:v);else contacts.push(c);$('editor').close();setDirty();if(duplicate)notice('שימו לב: קיים איש קשר עם שם או מספר זהה. בדקו כפילויות לפני שמירה.');});
$('add-contact').onclick=$('empty-add').onclick=()=>openEditor();
$('search').oninput=render;
$('save').onclick=async()=>{if(busy||!dirty)return;busy=true;render();try{saved=await api('/api/book','PUT',{revision:saved.revision,contacts});contacts=structuredClone(saved.contacts);dirty=false;$('degraded').classList.add('hidden');notice('השינויים נשמרו בהצלחה');}catch(e){notice(e.status===409?'הספר עודכן בחלון אחר. השינויים שלכם נשארו במסך. ייצאו או העתיקו אותם לפני טעינה מחדש.':e.message,true);}finally{busy=false;render();}};
$('reload').onclick=async()=>{if(dirty&&!await confirmAction('טעינה מחדש','השינויים שטרם נשמרו יבוטלו. להמשיך?','אישור טעינה'))return;await load();};
document.querySelectorAll('.close-dialog').forEach(b=>b.onclick=()=>b.closest('dialog').close());
function confirmAction(title,message,label){return new Promise(resolve=>{$('confirm-title').textContent=title;$('confirm-message').textContent=message;$('confirm-ok').textContent=label;const d=$('confirm-dialog');const done=value=>{d.close();d.oncancel=null;resolve(value);};$('confirm-ok').onclick=()=>done(true);$('confirm-cancel').onclick=()=>done(false);d.oncancel=e=>{e.preventDefault();done(false);};d.showModal();});}
$('import-csv').onclick=()=>{preview=null;$('csv-file').value='';$('import-content').replaceChildren();$('import-merge').classList.add('hidden');$('import-replace').classList.add('hidden');$('import-dialog').showModal();};
$('csv-file').onchange=async()=>{const file=$('csv-file').files[0];preview=null;$('import-merge').classList.add('hidden');$('import-replace').classList.add('hidden');if(!file)return;try{if(file.size>info.limits.csvBytes)throw new Error('הקובץ גדול ממגבלת הייבוא');$('import-content').textContent='מכין תצוגה מקדימה…';const bytes=new Uint8Array(await file.arrayBuffer());let binary='';for(let i=0;i<bytes.length;i+=8192)binary+=String.fromCharCode(...bytes.subarray(i,i+8192));preview=await api('/api/import/preview','POST',{data:btoa(binary)});const content=$('import-content');content.replaceChildren(element('h3','',`${preview.contacts.length} אנשי קשר · ${preview.encoding}`));for(const message of preview.warnings)content.append(element('p','warning',message));const scroll=element('div','preview-scroll'),table=element('table');for(const c of preview.contacts){const tr=element('tr');tr.append(element('td','',c.name),element('td','phone',c.phones.join(' · ')));table.append(tr);}scroll.append(table);content.append(scroll);$('import-merge').classList.remove('hidden');$('import-replace').classList.remove('hidden');}catch(e){$('import-content').textContent=e.message;}};
async function applyImport(replace){if(!preview)return;if(replace&&!await confirmAction('החלפת הספר','כל אנשי הקשר במסך יוחלפו ברשימת הייבוא. השינוי יישמר רק בלחיצה על שמירת שינויים.','אישור החלפה'))return;const incoming=structuredClone(preview.contacts);let added=0,skipped=0;const next=replace?[]:structuredClone(contacts);const key=c=>c.name.trim()+'\0'+c.phones.join('\0');const seen=new Set(next.map(key));for(const c of incoming){if(seen.has(key(c))){skipped++;continue;}seen.add(key(c));next.push(c);added++;}if(next.length>info.limits.contacts){notice('המיזוג חורג ממגבלת אנשי הקשר.',true);return;}contacts=next;$('import-dialog').close();setDirty();notice(`נוספו ${added} אנשי קשר לתצוגה. ${skipped} כפילויות זהות דולגו. יש לשמור את השינויים.`);}
$('import-merge').onclick=()=>applyImport(false);$('import-replace').onclick=()=>applyImport(true);
async function openBackups(){if(busy||!ready)return;$('backups-dialog').showModal();$('backup-list').textContent='טוען גיבויים…';try{const backups=await api('/api/backups');$('backup-list').replaceChildren();if(!backups.length)$('backup-list').textContent='עדיין אין גיבויים. גיבוי ראשון ייווצר לפני השמירה השנייה.';for(const b of backups){const row=element('div','backup-row'),label=element('div','',`גרסה ${b.revision}`);label.append(element('small','',formatDate(b.updatedAt)));const button=element('button','secondary','שחזור');button.onclick=async()=>{if(!await confirmAction('שחזור גיבוי',`לשחזר את גרסה ${b.revision}? השינויים הנוכחיים במסך יוחלפו והשחזור יישמר מיד כגרסה חדשה.`,'אישור שחזור'))return;button.disabled=true;busy=true;render();try{saved=await api('/api/restore','POST',{id:b.id,revision:saved.revision});contacts=structuredClone(saved.contacts);dirty=false;$('degraded').classList.add('hidden');notice('הגיבוי שוחזר ונשמר');}catch(e){notice(e.message,true);}finally{busy=false;button.disabled=false;render();}};row.append(label,button);$('backup-list').append(row);}}catch(e){$('backup-list').textContent=e.message;}}
$('nav-backups').onclick=openBackups;
function openHelp(){if(!info){notice('יש לטעון את הספר לפני הצגת פרטי החיבור.',true);return;}$('phone-urls').replaceChildren();if(!info.urls.length)$('phone-urls').textContent='לא נמצאה כתובת IPv4 בתת־הרשת המוגדרת. בדקו את חיבור הרשת.';for(const url of info.urls){const row=element('div','url-row'),code=element('code','',url);code.dir='ltr';const copy=element('button','secondary','העתקת כתובת');copy.onclick=async()=>{try{await navigator.clipboard.writeText(url);copy.textContent='הועתק';}catch{notice('לא ניתן להעתיק אוטומטית. סמנו והעתיקו את הכתובת.',true);}};row.append(code,copy);$('phone-urls').append(row);}$('help-dialog').showModal();}
$('nav-help').onclick=$('connection-help').onclick=openHelp;
$('nav-book').onclick=()=>$('search').focus();
window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});
window.addEventListener('keydown',e=>{if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='k'){e.preventDefault();$('search').focus();}});
load();
// Export the visible draft when dirty, including after a revision conflict.
// The server endpoint continues to export the last committed book for other clients.
document.querySelector('a[download]').addEventListener('click',event=>{
 if(!dirty)return;
 event.preventDefault();
 const escapeCell=value=>{
  let s=value;
  if(/^[=+\-@']/.test(s))s="'"+s;
  return '"'+s.replaceAll('"','""')+'"';
 };
 const rows=[['name','phone1','phone2','phone3'],...contacts.map(c=>[c.name,...c.phones,...Array(3-c.phones.length).fill('')])];
 const csv='\ufeff'+rows.map(row=>row.map(escapeCell).join(',')).join('\r\n')+'\r\n';
 const url=URL.createObjectURL(new Blob([csv],{type:'text/csv;charset=utf-8'}));
 const link=element('a');link.href=url;link.download='nimbus-phonebook-draft.csv';document.body.append(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000);
 notice('הטיוטה יוצאה לקובץ. השינויים עדיין לא נשמרו בספר המשותף.');
});

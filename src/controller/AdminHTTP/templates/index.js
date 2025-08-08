async function fetchKeys(id){
  const res = await fetch('/api/features/'+id+'/keys');
  if(!res.ok){ return [] }
  return await res.json();
}

async function fetchFeatures(){
  const res = await fetch('/api/features');
  const data = await res.json();
  const tbody = document.getElementById('features');
  tbody.innerHTML='';
  for(const f of (data||[])){
    const tr = document.createElement('tr');
    const keys = await fetchKeys(f.id);
    const keysHtml = (keys||[]).map(k=>
      '<div class="keys">'+
      '<b>'+k.key+'</b> = '+k.value+
      ' <button onclick="deleteKey(\''+k.id+'\')">✖</button>'+
      '</div>'
    ).join('');
    tr.innerHTML = '<td>' + f.name + '</td>'
      + '<td>' + (f.description||'') + '</td>'
      + '<td>' + f.value + '</td>'
      + '<td>' + keysHtml + '<div class="row"><input id="k-name-'+f.id+'" placeholder="key name" style="width:140px"/> '
      + '<input type="number" id="k-val-'+f.id+'" placeholder="0" style="width:80px"/> '
      + '<input id="k-desc-'+f.id+'" placeholder="description" style="width:160px"/> '
      + '<button onclick="createKey(\''+f.id+'\')">Add key</button></div></td>'
      + '<td>'
      + '  <input type="number" id="val-' + f.id + '" value="' + f.value + '" style="width:80px"/>'
      + '  <button onclick="setValue(\'' + f.id + '\')">Save</button>'
      + '  <button onclick="del(\'' + f.id + '\')">Delete</button>'
      + '</td>';
    tbody.appendChild(tr);
  }
}

async function createFeature(){
  const name = document.getElementById('f-name').value.trim();
  const description = document.getElementById('f-desc').value.trim();
  if(!name){return}
  const res = await fetch('/api/features',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name, description})});
  if(res.ok){ document.getElementById('f-name').value=''; document.getElementById('f-desc').value=''; await fetchFeatures(); }
}

async function setValue(id){
  const v = parseInt(document.getElementById('val-'+id).value||'0',10);
  const res = await fetch('/api/features/'+id+'/value',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({value:v})});
  if(res.ok){ await fetchFeatures(); }
}

async function del(id){
  if(!confirm('Delete feature?')) return;
  const res = await fetch('/api/features/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

async function createKey(featureId){
  const key = document.getElementById('k-name-'+featureId).value.trim();
  const value = parseInt(document.getElementById('k-val-'+featureId).value||'0',10);
  const description = document.getElementById('k-desc-'+featureId).value.trim();
  if(!key){return}
  const res1 = await fetch('/api/features/'+featureId+'/keys',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({key, description})});
  if(res1.ok){
    const id = (await res1.json()).id;
    await fetch('/api/keys/'+id+'/value',{method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({value})});
    await fetchFeatures();
  }
}

async function deleteKey(id){
  const res = await fetch('/api/keys/'+id,{method:'DELETE'});
  if(res.ok){ await fetchFeatures(); }
}

window.addEventListener('load', fetchFeatures);

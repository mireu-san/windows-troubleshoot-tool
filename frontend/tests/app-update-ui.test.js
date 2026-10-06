import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';
import { JSDOM, VirtualConsole } from 'jsdom';

const waitFor = async predicate => {
  for (let i = 0; i < 100; i++) {
    if (predicate()) return;
    await new Promise(resolve => setTimeout(resolve, 10));
  }
  assert.fail('Update UI did not reach expected state');
};

test('startup errors are visible and manual recheck recovers without losing a prepared update', async () => {
  const bundle = await build({root:fileURLToPath(new URL('..',import.meta.url)),configFile:false,logLevel:'silent',build:{write:false,minify:false}});
  const errors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on('jsdomError', error => errors.push(error));
  const dom = new JSDOM('<html><body><div id="app"></div></body></html>', {url:'http://localhost',runScripts:'dangerously',pretendToBeVisual:true,virtualConsole});
  const {window} = dom;
  let checks = 0, downloads = 0, resolveCheck, resolveDownload;
  window.go = {main:{App:{
    GetLanguageSettings:async()=>({preference:'ko',language:'ko'}),
    GetAppVersion:async()=> '2026.10.06', IsRunning:async()=>true,
    CheckAppUpdate:()=> {
      checks++;
      if (checks === 1) return Promise.reject(new Error('Network unavailable'));
      return new Promise(resolve=>{resolveCheck=resolve});
    },
    DownloadAppUpdate:()=> {downloads++; return new Promise(resolve=>{resolveDownload=resolve});},
  }}};
  window.runtime = {EventsOn:()=>{}};
  window.eval(bundle.output.find(item=>item.type==='chunk' && item.isEntry).code);
  const $ = selector=>window.document.querySelector(selector);
  try {
    await waitFor(()=>$('#appUpdateMessage')?.textContent.includes('Network unavailable'));
    assert.equal($('#appUpdateNotice').hidden, false);
    assert.equal($('#checkAppUpdate').disabled, false);
    $('#checkAppUpdate').click();
    assert.equal($('#checkAppUpdate').disabled, true);
    assert.match($('#appUpdateMessage').textContent, /확인하고/);
    $('#checkAppUpdate').click();
    assert.equal(checks, 2);
    resolveCheck({available:false,message:'현재 최신 버전을 사용하고 있습니다.'});
    await waitFor(()=>!$('#checkAppUpdate').disabled);
    assert.match($('#appUpdateMessage').textContent, /최신 버전/);
    $('#checkAppUpdate').click();
    resolveCheck({available:true,canDownload:true,message:'New version'});
    await waitFor(()=>downloads===1);
    assert.equal($('#checkAppUpdate').disabled, true);
    resolveDownload();
    await waitFor(()=>!$('#downloadAppUpdate').disabled);
    assert.equal($('#downloadAppUpdate').textContent, '업데이트 설치');
    assert.equal($('#checkAppUpdate').disabled, true);
    assert.deepEqual(errors, []);
  } finally {window.close();}
});

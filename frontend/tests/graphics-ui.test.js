import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';
import { JSDOM, VirtualConsole } from 'jsdom';

const waitFor = async predicate => {
  for (let i=0;i<100;i++) {
    if (predicate()) return;
    await new Promise(resolve => setTimeout(resolve,10));
  }
  assert.fail('UI did not reach expected state');
};

test('graphics UI boots after language init, routes updates, marks symptoms and saves reports', async () => {
  const bundle = await build({root:fileURLToPath(new URL('..',import.meta.url)),configFile:false,logLevel:'silent',build:{write:false,minify:false}});
  const code = bundle.output.find(item => item.type==='chunk' && item.isEntry).code;
  const errors = [];
  const virtualConsole = new VirtualConsole();
  virtualConsole.on('jsdomError',error => errors.push(error));
  const dom = new JSDOM('<!doctype html><html><body><div id="app"></div></body></html>',{url:'http://localhost',runScripts:'dangerously',pretendToBeVisual:true,virtualConsole});
  const {window} = dom;
  const calls = [], events = {};
  const report = {manufacturer:'Example',model:'Hybrid',from:'2026-10-05T12:00:00Z',until:'2026-10-05T12:02:00Z',gpus:[{name:'Intel GPU',vendor:'intel',version:'1',problemCode:0},{name:'NVIDIA GPU',vendor:'nvidia',version:'2'}],monitors:[],findings:[{title:'관련 오류 근거를 찾지 못했습니다',evidence:'<img src=x onerror=alert(1)>',action:'확인'}],issues:['partial collection'],events:[]};
  window.go={main:{App:{
    GetLanguageSettings:async()=>({preference:'ko',language:'ko'}),
    GetAppVersion:async()=> 'test',CheckAppUpdate:async()=>({available:false}),IsRunning:async()=>false,
    ScanGraphics:async()=>{calls.push('scan');return report},
    MarkGraphicsFlicker:async()=>{calls.push('mark');return {...report,marker:'2026-10-05T12:01:00Z'}},
    StartGraphicsUpdate:async vendor=>{calls.push(vendor);return {message:'제조사 업데이트 도구를 실행했습니다. 도구에서 설치를 마친 뒤 다시 검사해 주세요.'}},
    ExportGraphicsReport:async answers=>{calls.push(JSON.parse(answers));return 'C:\\reports\\graphics.json'},
    ExportRepairDiagnostics:async()=> 'C:\\reports\\repair.txt',
  }}};
  window.runtime={EventsOn:(name,handler)=>{events[name]=handler}};
  window.eval(code);
  const $=selector=>window.document.querySelector(selector);
  await waitFor(()=>$('#scanGraphicsButton'));
  $('#scanGraphicsButton').click();
  await waitFor(()=>$('#graphicsResults') && !$('#graphicsResults').hidden && !$('#scanGraphicsButton').disabled);
  assert.equal($('#graphicsDevices').querySelectorAll('article').length,2);
  assert.equal($('#graphicsDevices').querySelectorAll('button').length,2);
  assert.equal($('#graphicsFindings').querySelector('img'),null,'event strings must not become HTML');
  $('#graphicsDevices').querySelectorAll('button')[1].click();
  await waitFor(()=>calls.includes('nvidia')&&!$('#scanGraphicsButton').disabled);
  assert.match($('#graphicsStatus').textContent,/도구/);
  $('#markFlickerButton').click();
  await waitFor(()=>calls.includes('mark')&&!$('#scanGraphicsButton').disabled);
  $('#graphicsScope').value='external';
  $('#graphicsScope').dispatchEvent(new window.Event('change'));
  assert.match($('#graphicsAdvice').textContent,/케이블/);
  $('#saveGraphicsButton').click();
  await waitFor(()=>$('#graphicsStatus').textContent.includes('graphics.json'));
  assert.equal(calls.at(-1).scope,'external');
  events['repair:event']({stage:'error',debugAvailable:true,title:'Failed',message:'Error'});
  assert.equal($('#debugPanel').hidden,false);
  $('#exportDiagnostics').click();
  await waitFor(()=>$('#debugStatus').textContent.includes('repair.txt'));
  assert.deepEqual(errors,[]);
  window.close();
});

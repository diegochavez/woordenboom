// Run against `npm run web`. Uses isolated API fixtures; never writes learning data.
// PLAYWRIGHT_PATH may point to an existing Playwright installation.
const { chromium } = require(process.env.PLAYWRIGHT_PATH || 'playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const output = process.env.QA_OUTPUT_DIR || path.resolve(__dirname, '../../../docs/mobile-design/monochrome/screenshots');
fs.mkdirSync(output, { recursive: true });
const base = process.env.QA_BASE_URL || 'http://localhost:8081';
const topics = ['Daily Basics', 'Housing & Renting', 'Government & Municipality', 'Work & Office'].map((title, i) => ({
  title, slug: ['daily-basics','housing','government','work'][i], description: ['Everyday words for everyday conversations.','Feel at home in your new neighbourhood.','Appointments and local services.','Meetings, colleagues and working life.'][i], wordCount: 20, startedCount: 3, isAvailable: true,
})).concat([{title:'General vocabulary',slug:'general-vocabulary',description:'All words',wordCount:3,startedCount:0,isAvailable:true}]);
const questions = [
  { id:'q1', wordId:'w1', word:'gemeente', translation:'municipality', topic:'Government & Municipality', example:'De gemeente helpt je met je aanvraag.' },
  { id:'q2', wordId:'w2', word:'gezondheidszorg', translation:'healthcare', topic:'Daily Basics', example:'Goede gezondheidszorg is belangrijk.' },
  { id:'q3', wordId:'w3', word:'alsjeblieft', translation:'please', topic:'Daily Basics', example:'Een koffie, alsjeblieft.' },
];
(async () => {
  const browser = await chromium.launch({ headless: true, ...(process.env.CHROME_PATH ? { executablePath: process.env.CHROME_PATH } : {}) });
  const results=[];
  async function setup({auth=true,profile=true}={}) {
    const context=await browser.newContext({viewport:{width:375,height:812}});
    if(auth) await context.addInitScript(()=>{localStorage.setItem('woordenboom.accessToken','qa-fixture');localStorage.setItem('woordenboom.refreshToken','qa-fixture');});
    const state={answers:0,known:0,progressLoads:0,mapLoads:0,empty:false,failAnswer:false,failLoad:false,slow:false};
    await context.route('**/*', async route=>{
      const url=new URL(route.request().url());
      if(url.port!=='8080') return route.continue();
      const p=url.pathname; let body={}; let status=200;
      if(p==='/me') body={user:{id:'qa',displayName:'Alex',profileComplete:profile}};
      else if(p.endsWith('/review-timing')) body={rememberedCount:2,recommendedInterval:'3d',nextReviewAt:null,paused:false};
      else if(p==='/topics') body={topics,cefrLevels:[{level:'A1',isAvailable:true},{level:'A2',isAvailable:true}]};
      else if(p==='/topics/preview') body={wordCount:3,words:questions.map(q=>({...q,id:q.wordId}))};
      else if(p.endsWith('/words')&&p.startsWith('/topics/')) body={words:questions.map(q=>({...q,id:q.wordId}))};
      else if(p==='/progress/summary'){ state.progressLoads++; body={dailyDone:state.answers,dailyGoal:10,dueWords:13,masteredWords:3,topicsStarted:3,reviewedWords:72,totalWords:193,strengthScore:38}; }
      else if(p==='/knowledge-map') {state.mapLoads++;body={availableWords:193,knownWords:3+state.known,learningWords:69,totalWords:2000,ranges:Array.from({length:20},(_,i)=>({startRank:i*100+1,endRank:(i+1)*100,total:100,available:i<6?30:0,known:i<6?i*3:0,learning:i<6?i*5:0}))};}
      else if(p.startsWith('/knowledge-map/ranges/')) body={words:state.empty?[]:questions.map(q=>({...q,id:q.wordId,englishExample:'An example sentence.',cefrLevel:'A2'}))};
      else if(p==='/sessions/next') {state.lastTopicSlugs=route.request().postDataJSON().topicSlugs;if(state.failLoad){status=503;body={error:'unavailable'};} else {state.known=0;body={id:'test-session',questions:state.empty?[]:questions};}}
      else if(p.endsWith('/answer')||p.endsWith('/mark')) {
        if(state.failAnswer){status=503;body={error:'save_failed'};} else {state.answers++;const data=route.request().postDataJSON();if(data.answer==='Already Knew'||data.action==='already_knew')state.known++;body={correctCount:state.known};}
        if(state.slow) await new Promise(r=>setTimeout(r,350));
      } else if(p==='/auth/request-code')body={status:'sent'};
      else if(p==='/auth/verify-code')body={accessToken:'qa-fixture',refreshToken:'qa-fixture',user:{id:'qa',displayName:'Alex',profileComplete:true}};
      else if(p==='/profile')body={status:'saved'};
      await route.fulfill({status,contentType:'application/json',body:JSON.stringify(body)});
    });
    const page=await context.newPage();
    page.on('pageerror',e=>{console.error('Browser error:',e.message);results.push({error:e.message});});
    page.on('console',m=>{if(m.type()==='error')console.error('Console:',m.text().slice(0,1500));});
    return {page,context,state};
  }
  async function shot(page,name){await page.waitForFunction(() => [...document.querySelectorAll('[data-testid="word-reveal"]')].filter(el => el.getBoundingClientRect().width > 0).every(el => Number(getComputedStyle(el).opacity) >= 0.999));await page.screenshot({path:path.join(output,`${name}.png`),animations:'disabled',timeout:15000});}
  async function go(page,url,text){console.log('Checking',url);await page.goto(base+url); await page.getByText(text,{exact:true}).first().waitFor();}
  async function noOverflow(page,label){const over=await page.evaluate(()=>Array.from(document.querySelectorAll('*')).filter(el=>{const b=el.getBoundingClientRect();const s=getComputedStyle(el);return b.width>0&&s.position!=='absolute'&&(b.right>innerWidth+2||b.left<-2)}).map(el=>el.textContent?.slice(0,70)));assert.equal(over.length,0,`${label}: ${over.join(';')}`);}
  const {page,state}=await setup();
  await go(page,'/home','Your daily practice');
  await page.getByRole('link',{name:'Start practice',exact:true}).click();
  await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  assert.deepEqual(state.lastTopicSlugs,['general-vocabulary']);
  await go(page,'/review','gemeente');assert.deepEqual(state.lastTopicSlugs,['general-vocabulary']);
  await go(page,'/home','Your daily practice');
  await page.getByRole('link',{name:'Choose categories',exact:true}).click();
  await page.getByText('Choose your categories.',{exact:true}).filter({visible:true}).waitFor();
  results.push('Start practice and the default review route use General vocabulary; Choose categories opens the selector.');
  await page.getByRole('checkbox',{name:/General vocabulary/}).click();
  assert.equal(await page.getByRole('checkbox',{name:/General vocabulary/}).isChecked(),true);
  await page.getByRole('checkbox',{name:/Daily Basics/}).click();
  assert.equal(await page.getByRole('checkbox',{name:/General vocabulary/}).isChecked(),false);
  await page.getByRole('checkbox',{name:/Housing & Renting/}).click();
  await page.getByText('3 unique words available',{exact:true}).waitFor();
  await page.getByRole('checkbox',{name:/General vocabulary/}).click();
  assert.equal(await page.getByRole('checkbox',{name:/Daily Basics/}).isChecked(),false);
  assert.equal(await page.getByRole('checkbox',{name:/Housing & Renting/}).isChecked(),false);
  results.push('General selection clears the mix; specific categories clear General; overlapping preview counts unique IDs.');
  for(const width of [320,375,414,768]){
    await page.setViewportSize({width,height:812});
    await go(page,'/home','Your daily practice');await noOverflow(page,`home ${width}`);await shot(page,`01-dashboard-${width}`);await page.getByText('Learning strength',{exact:true}).scrollIntoViewIfNeeded();await shot(page,`01-dashboard-detail-${width}`);await noOverflow(page,`dashboard detail ${width}`);
    await go(page,'/topics','Choose your categories.');await noOverflow(page,`topics ${width}`);await shot(page,`02-categories-${width}`);
    await page.getByRole('checkbox',{name:/Daily Basics/}).click();await page.getByRole('button',{name:'Start practice',exact:true}).click();await page.getByText('gemeente',{exact:true}).filter({visible:true}).waitFor();
    assert.equal(await page.getByText('municipality',{exact:true}).filter({visible:true}).count(),0,'Translation must be hidden on front');
    const frontBox = await page.getByTestId('study-word').filter({visible:true}).boundingBox();
    await noOverflow(page,`word ${width}`);await shot(page,`03-study-front-${width}`);
    await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByText('municipality',{exact:true}).filter({visible:true}).waitFor();await noOverflow(page,`answer ${width}`);await shot(page,`04-study-back-${width}`);
    const backBox = await page.getByTestId('study-word').filter({visible:true}).boundingBox();
    assert.ok(Math.abs(frontBox.y - backBox.y) < 1, 'Flipping must preserve word height');
    await page.getByRole('button',{name:'Know this word',exact:true}).click();await page.getByText('gezondheidszorg',{exact:true}).filter({visible:true}).waitFor();await noOverflow(page,`long word ${width}`);await shot(page,`05-long-word-${width}`);
    const nextBox = await page.getByTestId('study-word').filter({visible:true}).boundingBox();
    assert.ok(Math.abs(frontBox.y - nextBox.y) < 1, 'Next word must preserve word height');
    for(const answer of ['Not yet, keep learning this word','Know this word']){await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:answer,exact:true}).click();}
    await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).waitFor();await noOverflow(page,`completion ${width}`);await shot(page,`06-stack-complete-${width}`);assert.match(await page.getByText('known',{exact:true}).filter({visible:true}).locator('..').innerText(),/2/);assert.match(await page.getByText('still learning',{exact:true}).filter({visible:true}).locator('..').innerText(),/1/);
    await page.getByRole('button',{name:'Practice again',exact:true}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
    await go(page,'/knowledge-map','Your Dutch, word by word.');await noOverflow(page,`map ${width}`);await shot(page,`07-map-${width}`);
    const rangeButton=page.getByRole('button',{name:/Words 1 to 100,/});
    const target=await rangeButton.boundingBox();
    const pill=await rangeButton.getByText('100',{exact:true}).locator('..').boundingBox();
    assert.ok(target.height>=44,'Range target must remain easy to tap');
    assert.ok(pill.height<=34,'Pill should be visually compact');
  }
  results.push('Main layouts passed at 320, 375, 414 and 768px; word position remains within 1px across flips and next cards.');
  await go(page,'/review?topic=daily-basics','gemeente');
  await page.emulateMedia({reducedMotion:'reduce'});
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    window.revealOpacities = [];
    window.revealObserver = new MutationObserver(() => {
      const el = [...document.querySelectorAll('[data-testid="word-reveal"]')].find(el=>el.getBoundingClientRect().width > 0);
      if(el) window.revealOpacities.push(Number(getComputedStyle(el).opacity));
    });
    window.revealObserver.observe(document.body,{attributes:true,childList:true,subtree:true});
  });
  await page.getByRole('button',{name:'Flip',exact:true}).click();
  await page.getByText('municipality',{exact:true}).filter({visible:true}).waitFor();
  assert.ok(await page.evaluate(()=>window.revealOpacities.length > 0 && window.revealOpacities.every(value=>value===1)), 'Reduce Motion must skip the fade');
  await page.emulateMedia({reducedMotion:'no-preference'});await page.waitForTimeout(100);
  await page.evaluate(()=>{window.revealOpacities=[];});
  await page.getByRole('button',{name:'Show Dutch',exact:true}).click();
  await page.waitForTimeout(220);
  assert.ok(await page.evaluate(()=>window.revealOpacities.some(value=>value<1)), 'Normal motion must include a fade');
  await page.evaluate(()=>window.revealObserver.disconnect());
  results.push('Fade is present with normal motion and skipped with Reduce Motion.');

  await page.setViewportSize({width:375,height:812});
  await go(page,'/home','Your daily practice');const loads=state.progressLoads;
  await page.getByText('Daily Basics',{exact:true}).filter({visible:true}).click();await page.getByRole('checkbox',{name:/Daily Basics/}).waitFor();assert.equal(await page.getByRole('checkbox',{name:/Daily Basics/}).isChecked(),true);
  await page.getByRole('button',{name:'Start practice'}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  state.failAnswer=true;await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Know this word',exact:true}).click();await page.getByText(/Your answer was not saved/).waitFor();assert.equal(await page.getByText('1 / 3',{exact:true}).filter({visible:true}).count(),1);await shot(page,'08-save-error');
  state.failAnswer=false;state.slow=true;const answers=state.answers;
  await page.getByRole('button',{name:'Know this word',exact:true}).evaluate(el=>{el.click();el.click();});
  await page.getByText('2 / 3',{exact:true}).filter({visible:true}).waitFor();assert.equal(state.answers,answers+1,'Double tap should save once');state.slow=false;
  for(let i=0;i<2;i++){await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Not yet, keep learning this word',exact:true}).click();}
  await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).waitFor();await page.getByRole('button',{name:'Back to dashboard'}).click();await page.getByText('Your daily practice',{exact:true}).filter({visible:true}).waitFor();assert.ok(state.progressLoads>loads,'Dashboard must refresh on return');
  assert.ok(!page.url().includes('/review'));results.push('Answer failure/retry, duplicate taps, completion, restart, category deep link and dashboard refresh passed.');
  state.empty=true;await go(page,'/review?topic=daily-basics','You’re up to date.');assert.equal(await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).count(),0);await shot(page,'09-empty-stack');
  await go(page,'/knowledge-range?start=1','No words yet.');assert.equal(await page.getByText('Range complete.',{exact:true}).filter({visible:true}).count(),0);state.empty=false;
  state.failLoad=true;await go(page,'/review?topic=daily-basics','Couldn’t load your stack.');await shot(page,'10-load-error');state.failLoad=false;await page.getByRole('button',{name:'Try again'}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  await go(page,'/knowledge-map','Your Dutch, word by word.');const maps=state.mapLoads;await page.getByRole('button',{name:/Words 1 to 100,/}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  for(let i=0;i<3;i++){await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Know this word',exact:true}).click();}
  await page.getByText('Range complete.',{exact:true}).filter({visible:true}).waitFor();await shot(page,'11-range-complete');await page.getByRole('button',{name:'Back to map'}).click();await page.getByText('Your Dutch, word by word.',{exact:true}).filter({visible:true}).waitFor();assert.ok(state.mapLoads>maps);
  results.push('Empty session/range, load retry, range completion and map refresh passed.');
  await go(page,'/disclaimer','OpenTaal attribution');await shot(page,'12-licenses');await noOverflow(page,'licenses');
  const login=await setup({auth:false});await go(login.page,'/login','A little Dutch, every day.');await shot(login.page,'13-login');await login.page.getByRole('textbox',{name:'Email',exact:true}).fill('design-qa@example.test');await login.page.getByRole('button',{name:'Send code'}).click();await login.page.getByRole('textbox',{name:'6-digit code'}).waitFor();await shot(login.page,'14-code');await login.page.getByRole('button',{name:'Use another email'}).click();assert.equal(await login.page.getByRole('textbox',{name:'Email',exact:true}).isEditable(),true);
  const onboarding=await setup({profile:false});await go(onboarding.page,'/onboarding','Make yourself at home.');await shot(onboarding.page,'15-onboarding');await noOverflow(onboarding.page,'onboarding');

  for(const width of [320,414,768]) {
    for(const [target,label] of [[login.page,'login'],[onboarding.page,'onboarding']]) {
      await target.setViewportSize({width,height:568});await noOverflow(target,`${label} ${width}`);await shot(target,`16-${label}-${width}`);
    }
  }
  await page.setViewportSize({width:320,height:568});
  await go(page,'/topics','Choose your categories.');
  for(const topic of topics.slice(0,3)) await page.getByRole('checkbox',{name:new RegExp(topic.title.replace(/[&]/g,'.'))}).click();
  assert.equal(await page.getByRole('checkbox',{name:/Work & Office/}).isDisabled(),true);
  await page.getByRole('button',{name:'Start practice'}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  state.slow=true;await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Know this word',exact:true}).click();await page.getByRole('button',{name:'Leave word stack'}).click();
  await page.getByText('Choose your categories.',{exact:true}).filter({visible:true}).waitFor();await page.getByRole('button',{name:'Start practice'}).click();await page.getByText('1 / 3',{exact:true}).filter({visible:true}).waitFor();
  await new Promise(resolve=>setTimeout(resolve,500));assert.equal(await page.getByText('1 / 3',{exact:true}).filter({visible:true}).count(),1);state.slow=false;
  for(let i=0;i<3;i++){await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Know this word',exact:true}).click();}
  await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).waitFor();await page.getByRole('button',{name:'Practice again'}).scrollIntoViewIfNeeded();await shot(page,'17-completion-short-screen');await noOverflow(page,'short completion');
  results.push('Login email recovery, verification form, onboarding, short screens, selection limit and pending-save navigation checked.');
  assert.equal(results.filter(result=>typeof result==='object').length,0,'No browser runtime errors expected');

  fs.writeFileSync(path.join(output,'../qa-results.json'),JSON.stringify(results,null,2));console.log(results);await browser.close();
})().catch(e=>{console.error(e);process.exit(1)});

// Exercise completion, delayed saves and foreground refresh with a disposable user.
const {chromium}=require(process.env.PLAYWRIGHT_PATH || 'playwright');
const {execFileSync}=require('node:child_process');
const crypto=require('node:crypto'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'../../..'),output=path.join(root,'docs/mobile-design/daily-progress');
const id=crypto.randomUUID(),email=`daily-qa-${id}@example.test`;
const sql=input=>execFileSync('docker',['compose','exec','-T','postgres','psql','-X','-At','-v','ON_ERROR_STOP=1','-U','woordenboom','-d','woordenboom'],{cwd:root,input,encoding:'utf8'}).trim();
const encode=value=>Buffer.from(JSON.stringify(value)).toString('base64url');
const unsigned=`${encode({alg:'HS256',typ:'JWT'})}.${encode({sub:id,email,exp:Math.floor(Date.now()/1000)+900})}`;
const token=unsigned+'.'+crypto.createHmac('sha256',process.env.QA_AUTH_SECRET || 'local-development-secret-change-before-production').update(unsigned).digest('base64url');
(async()=>{
 let browser,releaseAnswer,releaseTopics;
 try {
  fs.mkdirSync(output,{recursive:true});
  sql(`BEGIN; INSERT INTO users(id,email) VALUES('${id}','${email}'); INSERT INTO user_profiles(user_id,display_name,age_range) VALUES('${id}','Daily QA','25-34'); COMMIT;`);
  browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
  const context=await browser.newContext({viewport:{width:375,height:812}});
  await context.addInitScript(token=>{localStorage.setItem('woordenboom.accessToken',token);localStorage.setItem('woordenboom.refreshToken','local-qa-session');},token);
  const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
  const count=page.getByTestId('daily-practice-count').filter({visible:true});
  const expectCount=async n=>{await count.waitFor();await page.waitForFunction(n=>[...document.querySelectorAll('[data-testid="daily-practice-count"]')].some(el=>el.getBoundingClientRect().width>0&&el.textContent.trim()===`${n} / 10`),n);};
  const headers={Authorization:'Bearer '+token};
  // The summary should render while category data is still loading.
  const topicsGate=new Promise(resolve=>releaseTopics=resolve);
  await page.route('**/topics',async route=>{await topicsGate;await route.continue();},{times:1});
  await page.goto('http://localhost:8081/home');await expectCount(0);releaseTopics();
  await page.getByRole('link',{name:'Start practice',exact:true}).click();
  for(let i=1;i<=10;i++){
   await page.getByText(`${i} / 10`,{exact:true}).filter({visible:true}).waitFor();
   await page.getByRole('button',{name:'Flip',exact:true}).click();
   await page.getByRole('button',{name:'Know this word',exact:true}).click();
  }
  await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).waitFor();
  await page.getByRole('button',{name:'Back to dashboard',exact:true}).click();await expectCount(10);
  // Return home before an answer commits. Its eventual success must invalidate home.
  await page.getByRole('link',{name:'Start practice',exact:true}).click();
  await page.getByRole('button',{name:'Flip',exact:true}).click();
  let started;const answerStarted=new Promise(resolve=>started=resolve),gate=new Promise(resolve=>releaseAnswer=resolve);
  await page.route('**/sessions/*/answer',async route=>{started();await gate;const response=await route.fetch();await route.fulfill({response});},{times:1});
  await page.getByRole('button',{name:'Know this word',exact:true}).click();await answerStarted;
  const homeRead=page.waitForResponse(r=>r.url().endsWith('/progress/summary'));
  await page.getByRole('button',{name:'Leave word stack'}).click();await homeRead;await expectCount(10);
  releaseAnswer();await expectCount(11);assert.ok(page.url().endsWith('/home'));
  // A successful summary survives a failed secondary request; resume triggers a read.
  const externalID=sql("SELECT bw.word_id FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' AND bw.batch_rank=9499;");
  const saved=await context.request.post(`http://localhost:8080/knowledge-map/words/${externalID}/mark`,{headers,data:{action:'should_learn'}});assert.equal(saved.status(),200);
  await page.route('**/topics',route=>route.fulfill({status:503,json:{error:'fixture_failure'}}),{times:1});
  await page.evaluate(()=>{Object.defineProperty(document,'visibilityState',{configurable:true,value:'hidden'});document.dispatchEvent(new Event('visibilitychange'));Object.defineProperty(document,'visibilityState',{configurable:true,value:'visible'});document.dispatchEvent(new Event('visibilitychange'));});
  await expectCount(12);await page.getByRole('button',{name:'Retry refresh'}).waitFor();
  await page.getByRole('button',{name:'Retry refresh'}).click();await page.getByRole('button',{name:'Retry refresh'}).waitFor({state:'hidden'});
  // Finish the last one-word map range and return through the map to home.
  await page.goto('http://localhost:8081/knowledge-range?start=9500');
  await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Know this word',exact:true}).click();
  await page.getByText('Range complete.',{exact:true}).filter({visible:true}).waitFor();
  await page.getByRole('button',{name:'Back to map'}).click();await page.getByRole('link',{name:'Back',exact:true}).click();await expectCount(13);
  // Open the 1200 range, then use both back arrows to return home.
  await page.getByRole('link',{name:'Open knowledge map'}).click();
  await page.getByRole('button',{name:/Words 1101 to 1200/}).waitFor();
  await page.screenshot({path:path.join(output,'knowledge-map.png')});
  await page.getByRole('button',{name:/Words 1101 to 1200/}).click();
  await page.getByRole('button',{name:'Leave word stack'}).click();
  await page.getByRole('link',{name:'Back',exact:true}).click();await expectCount(13);
  const summary=await context.request.get('http://localhost:8080/progress/summary',{headers});assert.match(summary.headers()['cache-control'],/no-store/);assert.equal((await summary.json()).dailyDone,13);
  await page.screenshot({path:path.join(output,'updated-dashboard.png')});
  assert.deepEqual(errors,[]);
  fs.writeFileSync(path.join(output,'checks.json'),JSON.stringify({sessionCompletion:10,delayedSave:11,foregroundMapSave:12,mapCompletion:13,range1200BackNavigation:true,slowCategoriesDoNotBlock:true,secondaryFailurePreservesSummary:true,retryRefresh:true,noStore:true,browserErrors:errors},null,2)+'\n');
  console.log('Daily progress QA passed: 0 → 10 after completion → 11 after delayed save → 12 on foreground → 13 after map completion. Slow/failed secondary loads, retry and no-store verified.');
 } finally {releaseAnswer?.();releaseTopics?.();if(browser)await browser.close();sql(`DELETE FROM users WHERE id='${id}';`);}
})().catch(error=>{console.error(error);process.exitCode=1;});

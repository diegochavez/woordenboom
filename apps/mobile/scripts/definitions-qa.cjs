// Live content integration with a disposable local account; no email.
const {chromium}=require(process.env.PLAYWRIGHT_PATH || 'playwright');
const {execFileSync}=require('node:child_process');
const crypto=require('node:crypto'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'../../..'),output=path.join(root,'docs/content-review');
const id=crypto.randomUUID(),email=`definitions-qa-${id}@example.test`;
const sql=input=>execFileSync('docker',['compose','exec','-T','postgres','psql','-X','-At','-v','ON_ERROR_STOP=1','-U','woordenboom','-d','woordenboom'],{cwd:root,input,encoding:'utf8'}).trim();
const encode=value=>Buffer.from(JSON.stringify(value)).toString('base64url');
const unsigned=`${encode({alg:'HS256',typ:'JWT'})}.${encode({sub:id,email,exp:Math.floor(Date.now()/1000)+900})}`;
const token=unsigned+'.'+crypto.createHmac('sha256',process.env.QA_AUTH_SECRET || 'local-development-secret-change-before-production').update(unsigned).digest('base64url');
(async()=>{
 let browser;
 try{
  fs.mkdirSync(output,{recursive:true});
  sql(`BEGIN; INSERT INTO users(id,email) VALUES('${id}','${email}'); INSERT INTO user_profiles(user_id,display_name,age_range) VALUES('${id}','Definition QA','25-34'); COMMIT;`);
  browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
  const context=await browser.newContext({viewport:{width:375,height:812}});
  await context.addInitScript(token=>{localStorage.setItem('woordenboom.accessToken',token);localStorage.setItem('woordenboom.refreshToken','local-qa-session');},token);
  const page=await context.newPage(), errors=[];page.on('pageerror',e=>errors.push(e.message));
  const visible=locator=>locator.filter({visible:true});
  async function flip(){await visible(page.getByRole('button',{name:'Flip',exact:true})).click();await visible(page.getByRole('button',{name:'Know this word',exact:true})).waitFor();await page.waitForFunction(()=>[...document.querySelectorAll('[data-testid="word-reveal"]')].filter(el=>el.getBoundingClientRect().width>0).every(el=>Number(getComputedStyle(el).opacity)===1));}
  const next=page.waitForResponse(r=>r.url().endsWith('/sessions/next')&&r.status()===200);
  await page.goto('http://localhost:8081/review?topics=general-vocabulary');
  const session=await(await next).json(),first=session.questions[0];assert.ok(first.meanings?.length);
  await visible(page.getByTestId('study-word')).waitFor();
  assert.equal(await visible(page.getByTestId('dutch-definitions')).count(),0);
  const before=await visible(page.getByTestId('study-word')).boundingBox();
  await flip();await visible(page.getByText(first.meanings[0].dutch,{exact:true})).waitFor();
  const after=await visible(page.getByTestId('study-word')).boundingBox();assert.ok(Math.abs(before.y-after.y)<1);
  await visible(page.getByText(first.meanings[0].dutch,{exact:true})).scrollIntoViewIfNeeded();
  await page.screenshot({path:path.join(output,'practice-definition.png')});
  await visible(page.getByRole('button',{name:'Know this word',exact:true})).click();await visible(page.getByText('2 / 10',{exact:true})).waitFor();
  const lerenRank=Number(sql("SELECT bw.batch_rank FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id JOIN words w ON w.id=bw.word_id WHERE b.slug='release-9500' AND w.lemma='leren';"));
  for(const width of [320,375,414]){
   await page.setViewportSize({width,height:width===320?568:896});
   const range=page.waitForResponse(r=>r.url().endsWith(`/knowledge-map/ranges/${lerenRank}`)&&r.status()===200);
   await page.goto(`http://localhost:8081/knowledge-range?start=${lerenRank}`);
   const word=(await(await range).json()).words[0];assert.equal(word.meanings.length,2);
   await flip();
   for(const meaning of word.meanings) await visible(page.getByText(meaning.dutch,{exact:true})).scrollIntoViewIfNeeded();
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);
   assert.ok((await visible(page.getByRole('button',{name:'Know this word',exact:true})).boundingBox()).y< (width===320?568:896));
   await page.screenshot({path:path.join(output,`multiple-meanings-${width}.png`)});
  }
  const missing=Number(sql("SELECT bw.batch_rank FROM word_learning_content lc JOIN vocabulary_batch_words bw USING(batch_id,word_id) WHERE jsonb_array_length(lc.meanings)=0 ORDER BY bw.batch_rank LIMIT 1;"));
  await page.goto(`http://localhost:8081/knowledge-range?start=${missing}`);await flip();
  assert.equal(await visible(page.getByTestId('dutch-definitions')).count(),0);
  await visible(page.getByRole('button',{name:'Know this word',exact:true})).click();await visible(page.getByText('2 / 100',{exact:true})).waitFor();
  assert.deepEqual(errors,[]);
  fs.writeFileSync(path.join(output,'browser-checks.json'),JSON.stringify({livePractice:true,hiddenBeforeFlip:true,stableWordPosition:true,pairedDefinitions:true,widths:[320,375,414],scrolling:true,missingDefinitionFallback:true,answerSaves:true,browserErrors:errors},null,2)+'\n');
  console.log('Definition QA passed: live practice/map, hidden before flip, stable word, paired meanings, 320/375/414px, missing-definition fallback and saved answers.');
 }finally{if(browser)await browser.close();sql(`DELETE FROM users WHERE id='${id}';`);}
})().catch(error=>{console.error(error);process.exitCode=1;});

// Local integration QA with an isolated account, removed on exit. No emails.
const { chromium }=require(process.env.PLAYWRIGHT_PATH || 'playwright');
const { execFileSync }=require('node:child_process');
const crypto=require('node:crypto'), assert=require('node:assert/strict'), fs=require('node:fs'), path=require('node:path');
const root=path.resolve(__dirname,'../../..'), output=path.join(root,'docs/review-timing');
const id=crypto.randomUUID(), email=`timing-qa-${id}@example.test`;
const sql=input=>execFileSync('docker',['compose','exec','-T','postgres','psql','-X','-At','-v','ON_ERROR_STOP=1','-U','woordenboom','-d','woordenboom'],{cwd:root,input,encoding:'utf8'}).trim();
const encode=value=>Buffer.from(JSON.stringify(value)).toString('base64url');
const unsigned=`${encode({alg:'HS256',typ:'JWT'})}.${encode({sub:id,email,exp:Math.floor(Date.now()/1000)+900})}`;
const token=unsigned+'.'+crypto.createHmac('sha256',process.env.QA_AUTH_SECRET || 'local-development-secret-change-before-production').update(unsigned).digest('base64url');
const headers={Authorization:`Bearer ${token}`,'Content-Type':'application/json'};
async function api(endpoint,body){const res=await fetch('http://localhost:8080'+endpoint,{headers,...(body?{method:'POST',body:JSON.stringify(body)}:{})});assert.equal(res.status,200);return res.json();}
function approximatelyDue(timing,minutes){assert.ok(timing.nextReviewAt);const delta=(Date.parse(timing.nextReviewAt)-Date.now())/60000;assert.ok(Math.abs(delta-minutes)<1,`${delta} vs ${minutes}`);}
(async()=>{
 let browser;
 try {
  fs.mkdirSync(output,{recursive:true});
  sql(`BEGIN; INSERT INTO users(id,email) VALUES('${id}','${email}'); INSERT INTO user_profiles(user_id,display_name,age_range) VALUES('${id}','Review QA','25-34'); COMMIT;`);
  browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
  const context=await browser.newContext({viewport:{width:375,height:812}});
  await context.addInitScript(token=>{localStorage.setItem('woordenboom.accessToken',token);localStorage.setItem('woordenboom.refreshToken','local-qa-session');},token);
  const page=await context.newPage(), errors=[];page.on('pageerror',e=>errors.push(e.message));
  const response=page.waitForResponse(r=>r.url().endsWith('/sessions/next')&&r.status()===200);
  await page.goto('http://localhost:8081/review?topics=general-vocabulary');
  const session=await (await response).json();
  const timing=id=>api(`/words/${id}/review-timing`);
  async function flip(){await page.getByRole('button',{name:'Flip',exact:true}).click();await page.getByRole('button',{name:'Change review timing',exact:true}).waitFor();await page.waitForFunction(()=>{const el=document.querySelector('[data-testid="word-reveal"]');return el&&Number(getComputedStyle(el).opacity)===1;});}
  async function open(){await page.getByRole('button',{name:'Change review timing',exact:true}).click();await page.getByText('Next review',{exact:true}).waitFor();await page.getByRole('radio',{name:'Pause reviews',exact:true}).waitFor();await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));}
  async function choose(label,confirm='Remember & continue'){await page.getByRole('radio',{name:new RegExp('^'+label+'(, recommended)?$')}).click();await page.getByRole('button',{name:confirm,exact:true}).click();}
  await flip();await page.getByText('Review in 1 day',{exact:true}).waitFor();
  await page.screenshot({path:path.join(output,'01-card.png')});
  await open();await page.screenshot({path:path.join(output,'02-options.png')});
  await page.getByRole('radio',{name:'In 3 days',exact:true}).click();await page.getByRole('button',{name:'Cancel',exact:true}).click();
  assert.equal((await timing(session.questions[0].wordId)).rememberedCount,0);
  await open();await choose('In a week');await page.getByText('2 / 10',{exact:true}).waitFor();
  const first=await timing(session.questions[0].wordId);assert.equal(first.rememberedCount,1);approximatelyDue(first,10080);
  // A repeated answer with a different timing must not reschedule the original answer.
  await api(`/sessions/${session.id}/answer`,{questionId:session.questions[0].id,wordId:session.questions[0].wordId,answer:'Already Knew',reviewInterval:'paused'});
  assert.deepEqual(await timing(session.questions[0].wordId),first);
  await flip();await open();await page.getByRole('radio',{name:'Pause reviews',exact:true}).click();
  await page.screenshot({path:path.join(output,'03-pause.png')});
  await page.getByRole('button',{name:'Remember & pause',exact:true}).click();await page.getByText('3 / 10',{exact:true}).waitFor();
  const pausedID=session.questions[1].wordId;let paused=await timing(pausedID);assert.equal(paused.paused,true);assert.equal(paused.nextReviewAt,null);
  await flip();await page.getByRole('button',{name:'Not yet, keep learning this word',exact:true}).click();await page.getByText('4 / 10',{exact:true}).waitFor();
  approximatelyDue(await timing(session.questions[2].wordId),10);
  // Manual map browsing exposes paused words and allows resuming their schedule.
  assert.match(pausedID,/^[a-f0-9-]+$/);
  const rank=Number(sql(`SELECT bw.batch_rank FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' AND bw.word_id='${pausedID}';`));
  await page.goto(`http://localhost:8081/knowledge-range?start=${rank}`);await flip();await page.getByText('Reviews paused',{exact:true}).waitFor();await open();await choose('In 3 months');
  await page.getByText('2 / 100',{exact:true}).waitFor();paused=await timing(pausedID);assert.equal(paused.paused,false);assert.equal(paused.rememberedCount,2);approximatelyDue(paused,129600);
  // Short-screen layout, scrolling to the final option, and cancel via keyboard.
  await page.setViewportSize({width:320,height:568});await flip();await open();
  await page.getByRole('radio',{name:'Pause reviews',exact:true}).scrollIntoViewIfNeeded();
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);
  await page.screenshot({path:path.join(output,'04-small-screen.png')});
  await page.keyboard.press('Escape');await page.getByText('Next review',{exact:true}).waitFor({state:'hidden'});
  // An answer failure leaves the selected timing available for retry.
  await page.setViewportSize({width:414,height:896});await open();
  await page.route('**/knowledge-map/words/*/mark',route=>route.fulfill({status:503,contentType:'application/json',body:'{"error":"test_failure"}'}),{times:1});
  const yearID=sql(`SELECT bw.word_id FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' AND bw.batch_rank=${rank+1};`);
  await choose('In a year');await page.getByText(/Your answer was not saved/).waitFor();await page.getByText('Review in a year',{exact:true}).waitFor();
  await page.screenshot({path:path.join(output,'06-year-retry.png')});
  await page.getByRole('button',{name:'Know this word',exact:true}).click();await page.getByText('3 / 100',{exact:true}).waitFor();
  approximatelyDue(await timing(yearID),525600);
  // The server's capped recommendation is usable in the picker for mature words.
  sql(`UPDATE user_word_progress SET review_streak=20 WHERE user_id='${id}' AND word_key='${yearID}';`);
  await page.goto(`http://localhost:8081/knowledge-range?start=${rank+1}`);await flip();
  await page.getByText('Review in a year',{exact:true}).waitFor();await open();
  const yearly=page.getByRole('radio',{name:'In a year, recommended',exact:true});
  assert.equal(await yearly.getAttribute('aria-checked'),'true');await yearly.scrollIntoViewIfNeeded();
  await page.screenshot({path:path.join(output,'07-year-recommendation.png')});
  await page.getByRole('button',{name:'Cancel',exact:true}).click();
  // No early repeats when a whole category is scheduled later.
  sql(`INSERT INTO user_word_progress(user_id,word_key,topic_slug,correct_count,next_review_at,last_seen_at) SELECT '${id}',bw.word_id::text,bw.topic_slug,1,now()+interval '1 day',now() FROM vocabulary_topic_memberships bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' AND bw.topic_slug='immigration-documents' ON CONFLICT(user_id,word_key) DO UPDATE SET next_review_at=now()+interval '1 day';`);
  await page.goto('http://localhost:8081/review?topics=immigration-documents');await page.getByText('You’re up to date.',{exact:true}).waitFor();
  await page.screenshot({path:path.join(output,'05-up-to-date.png')});
  assert.deepEqual(errors,[]);
  fs.writeFileSync(path.join(output,'checks.json'),JSON.stringify({recommendedTiming:true,manualWeek:true,manualThreeMonths:true,manualYear:true,cappedYearRecommendation:true,cancelDoesNotSave:true,duplicateAnswerDoesNotReschedule:true,pauseAndResume:true,notYetTenMinutes:true,smallScreen320:true,escapeCloses:true,failedSaveRetry:true,futureReviewsNotRepeated:true,browserErrors:errors},null,2)+'\n');
  console.log('Review timing QA passed: recommended/week/month/year timing, cancel, duplicate answers, pause/resume, ten-minute retry, 320px sheet, Escape, failed-save retry, up-to-date state.');
 } finally {if(browser)await browser.close();sql(`DELETE FROM users WHERE id='${id}';`);}
})().catch(error=>{console.error(error);process.exitCode=1;});

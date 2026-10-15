// Live local-backend QA. Creates and removes one isolated test user; sends no email.
const { chromium } = require(process.env.PLAYWRIGHT_PATH || 'playwright');
const { execFileSync } = require('node:child_process');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const root = path.resolve(__dirname, '../../..');
const out = path.join(root, 'docs/vocabulary-9500');
const id = crypto.randomUUID();
const email = `vocabulary-qa-${id}@example.test`;
const sql = input => execFileSync('docker', ['compose','exec','-T','postgres','psql','-X','-v','ON_ERROR_STOP=1','-U','woordenboom','-d','woordenboom'], {cwd:root,input,stdio:['pipe','pipe','pipe']});
const encode = value => Buffer.from(JSON.stringify(value)).toString('base64url');
const unsigned = `${encode({alg:'HS256',typ:'JWT'})}.${encode({sub:id,email,exp:Math.floor(Date.now()/1000)+600})}`;
const token = unsigned+'.'+crypto.createHmac('sha256',process.env.QA_AUTH_SECRET || 'local-development-secret-change-before-production').update(unsigned).digest('base64url');
(async()=>{
  let browser;
  fs.mkdirSync(out,{recursive:true});
  try {
    sql(`BEGIN; INSERT INTO users(id,email) VALUES('${id}','${email}'); INSERT INTO user_profiles(user_id,display_name,age_range) VALUES('${id}','Vocabulary QA','25-34'); COMMIT;`);
    const headers={Authorization:`Bearer ${token}`};
    const map=await fetch('http://localhost:8080/knowledge-map',{headers}).then(r=>r.json());
    assert.equal(map.availableWords,9500);assert.equal(map.ranges.length,95);
    const last=await fetch('http://localhost:8080/knowledge-map/ranges/9401',{headers}).then(r=>r.json());
    assert.equal(last.words.length,100);assert.ok(last.words.every(w=>w.translation));
    browser=await chromium.launch({headless:true,...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{})});
    const context=await browser.newContext({viewport:{width:375,height:812}});
    await context.addInitScript(token=>{localStorage.setItem('woordenboom.accessToken',token);localStorage.setItem('woordenboom.refreshToken','local-qa-session');},token);
    const page=await context.newPage();
    const errors=[];page.on('pageerror',e=>errors.push(e.message));
    await page.goto('http://localhost:8081/knowledge-map');
    await page.getByText('9,500',{exact:true}).first().waitFor();
    await page.screenshot({path:path.join(out,'map.png'),fullPage:true});
    for(const width of [320,414]){
      await page.setViewportSize({width,height:812});
      await page.goto('http://localhost:8081/knowledge-range?start=9401');
      await page.getByText('1 / 100',{exact:true}).filter({visible:true}).waitFor();
      await page.getByRole('button',{name:'Flip',exact:true}).click();
      await page.getByText(last.words[0].translation,{exact:true}).filter({visible:true}).waitFor();
      await page.waitForFunction(()=>{const el=document.querySelector('[data-testid="word-reveal"]');return el && Number(getComputedStyle(el).opacity)===1;});
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1),false);
      await page.screenshot({path:path.join(out,`translation-${width}.png`),fullPage:true});
    }
    await page.getByRole('button',{name:'Know this word',exact:true}).click();
    await page.getByText('2 / 100',{exact:true}).filter({visible:true}).waitFor();
    // Exercise the actual category -> practice -> completion -> retry flow.
    await page.goto('http://localhost:8081/topics?topic=general-vocabulary');
    const nextResponse=()=>page.waitForResponse(r=>new URL(r.url()).pathname==='/sessions/next' && r.request().method()==='POST' && r.status()===200);
    let sessionResponse=nextResponse();
    await page.getByRole('button',{name:'Start practice',exact:true}).click();
    let session=await (await sessionResponse).json();
    const practiceWords=new Set();
    for(let stack=0;stack<3;stack++){
      await page.getByText('1 / 10',{exact:true}).filter({visible:true}).waitFor();
      assert.equal(session.questions.length,10);
      for(const q of session.questions){assert.equal(practiceWords.has(q.wordId),false,`Repeated ${q.word} in stack ${stack+1}`);practiceWords.add(q.wordId);}
      if(stack===2)break;
      for(let i=0;i<10;i++){
        await page.getByText(`${i+1} / 10`,{exact:true}).filter({visible:true}).waitFor();
        await page.getByRole('button',{name:'Flip',exact:true}).click();
        await page.getByRole('button',{name:stack===0?'Know this word':'Not yet, keep learning this word',exact:true}).click();
      }
      await page.getByText('Stack complete.',{exact:true}).filter({visible:true}).waitFor();
      sessionResponse=nextResponse();
      await page.getByRole('button',{name:'Practice again',exact:true}).click();
      session=await (await sessionResponse).json();
    }
    assert.equal(practiceWords.size,30);
    await page.goto('http://localhost:8081/disclaimer');
    await page.getByText('Translations and Dutch definitions',{exact:true}).waitFor();
    await page.screenshot({path:path.join(out,'sources.png'),fullPage:true});
    assert.deepEqual(errors,[]);
    fs.writeFileSync(path.join(out,'checks.json'),JSON.stringify({availableWords:map.availableWords,ranges:map.ranges.length,lastRangeWords:last.words.length,widths:[320,414],markAndAdvance:true,practiceStacks:3,uniquePracticeWords:practiceWords.size,browserErrors:errors},null,2)+'\n');
    console.log('Live backend and mobile QA passed: 9,500 words, 95 ranges, translation-only cards at 320/414px, mark/advance, 30 distinct practice words over three stacks, source notices.');
  } finally {
    if(browser) await browser.close();
    sql(`DELETE FROM users WHERE id='${id}';`);
  }
})().catch(e=>{console.error(e);process.exitCode=1;});

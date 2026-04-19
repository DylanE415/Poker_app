document.addEventListener('DOMContentLoaded', () => {
  const $ = (id) => document.getElementById(id);
  const statusEl = $('status'), whoEl = $('who');
  const serverMsgEl = $('serverMsg');

  const EMOTES = [
    { type: 'angle', label: 'Angle' },
    { type: 'sick',  label: 'So Sick' },
    { type: 'itsPoker', label: "It's Poker Phil" },
  ];

  let currentQueued = '';
  let lastHandCurrentBet = 0;
  let lastMinRaiseAdd    = 0;
  let lastHeroWas67 = false;
  let lastBoardLen = 0;

  let currentRaiseMin = 0;
  let currentRaiseMax = 100;

  function showMsg(text, type = '') {
    serverMsgEl.textContent = text;
    serverMsgEl.className = 'server-message' + (type ? ' ' + type : '') + (text ? ' show' : '');
  }

  const chipSound = $('chipSound');
  const checkSound = $('checkSound');
  const audio67 = $('audio67');
  const dealSound = $('dealSound');

  let audioUnlocked = false;
  function unlockAudioOnce() {
    if (audioUnlocked) return; audioUnlocked = true;
    try {
      const p1 = chipSound.play(); if (p1?.then) p1.then(()=>{ chipSound.pause(); chipSound.currentTime = 0; }).catch(()=>{});
      const p2 = checkSound.play(); if (p2?.then) p2.then(()=>{ checkSound.pause(); checkSound.currentTime = 0; }).catch(()=>{});
      const p3 = audio67.play(); if (p3?.then) p3.then(()=>{ audio67.pause(); audio67.currentTime = 0; }).catch(()=>{});
      const p4 = dealSound.play(); if (p4?.then) p4.then(()=>{ dealSound.pause(); dealSound.currentTime = 0; }).catch(()=>{});
    } catch {}
  }
  ['click','touchstart','keydown'].forEach(ev => document.addEventListener(ev, unlockAudioOnce, { once:true, passive:true }));
  function safePlayChip(){ try{ chipSound.currentTime=0; chipSound.play().catch(()=>{});}catch{} }
  function safePlayCheck(){ try{ checkSound.currentTime=0; checkSound.play().catch(()=>{});}catch{} }
  function safePlay67(){ try{ audio67.currentTime=0; audio67.play().catch(()=>{});}catch{} }
  function safePlayDeal(){ try{ dealSound.currentTime=0; dealSound.play().catch(()=>{});}catch{} }

  function playEmoteAudio(path){
    if (!path) return;
    try { new Audio(path).play().catch(()=>{}); } catch {}
  }

  let myName = '';
  (async function loadMe(){
    try {
      const res = await fetch('/me', { credentials: 'include' });
      if (res.status === 401) { location.href = '/login'; return; }
      if (!res.ok) return;
      const me = await res.json();
      myName = String(me?.username || '').trim();
      whoEl.textContent = myName ? `Signed in as ${myName}` : '';
    } catch {}
  })();

  async function doFetch(url, opts) { return await fetch(url, Object.assign({ credentials:'include' }, opts||{})); }
  const api = {
    async join(room, stack) {
      const jsonTry = await doFetch(`/join?room=${encodeURIComponent(room)}`, {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({ stack: Number(stack)||0 })
      });
      if (jsonTry.ok || jsonTry.status === 401) return jsonTry;
      const formTry = await doFetch(`/join?room=${encodeURIComponent(room)}`, {
        method:'POST', headers:{'Content-Type':'application/x-www-form-urlencoded'},
        body: new URLSearchParams({ stack: String(Number(stack)||0) }).toString()
      });
      if (formTry.ok || formTry.status === 401) return formTry;
      return await doFetch(`/join?room=${encodeURIComponent(room)}&stack=${encodeURIComponent(Number(stack)||0)}`);
    },
    async leave(room) { return await doFetch(`/leave?room=${encodeURIComponent(room)}`, { method:'POST' }); },
    async state(room) { return await doFetch(`/state?room=${encodeURIComponent(room)}`, { method:'GET' }); },
    async action(room, action, amount=0) {
      return await doFetch(`/action?room=${encodeURIComponent(room)}`, {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({ action, amount: Number(amount)||0 })
      });
    },
    async emote(room, emoteType) {
      return await doFetch(`/emote?room=${encodeURIComponent(room)}`, {
        method:'POST', headers:{'Content-Type':'application/json'},
        body: JSON.stringify({ emoteType })
      });
    },
    async sit(room, sitIn) {
      return await doFetch(`/sitInOrOut?room=${encodeURIComponent(room)}&sitIn=${sitIn}`, { method:'POST' });
    },
    async showHand(room, show) {
      const params = new URLSearchParams({ room: room, showHand: String(!!show) });
      return await doFetch(`/showHand?${params.toString()}`, { method:'POST' });
    },
    async logout(){ location.href='/login'; }
  };

  const roomSel = $('room'), stackInp = $('stack'),
        joinBtn = $('joinBtn'), leaveBtn = $('leaveBtn'),
        sitInBtn = $('sitInBtn'), sitOutBtn = $('sitOutBtn'),
        checkBtn = $('checkBtn'), callBtn = $('callBtn'), raiseBtn = $('raiseBtn'),
        raiseAmt = $('raiseAmt'), foldBtn = $('foldBtn'), clearBtn = $('clearBtn'),
        ledgerBtn = $('ledgerBtn'), raiseSlider = $('raiseSlider'),
        raiseAmtLabel = $('raiseAmtLabel'), callAmtEl = $('callAmt'),
        raiseUp = $('raiseUp'), raiseDown = $('raiseDown'),
        emoteBtn = $('emoteBtn'), emoteMenu = $('emoteMenu'),
        emoteCooldownEl = $('emoteCooldown'),
        timebankDisplayEl = $('timebankDisplay'),
        showHandBtn = $('showHandBtn'), hideHandBtn = $('hideHandBtn');
  const queuedLabel = $('queuedLabel'), queuedValue = $('queuedValue');

  $('logoutBtn').addEventListener('click', api.logout);
  function pulse(el){ if(!el) return; el.classList.remove('pulse'); void el.offsetWidth; el.classList.add('pulse'); setTimeout(()=>el.classList.remove('pulse'), 180); }
  function setQueuedDisplay(text){
    currentQueued = text || '';
    queuedValue.textContent = text || '—';
    queuedLabel.style.display='inline';
  }

  function buildEmoteMenu(){
    emoteMenu.innerHTML = '';
    for (const e of EMOTES) {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'emote-item';
      btn.setAttribute('role', 'menuitem');
      btn.dataset.emoteType = e.type;
      btn.textContent = e.label;
      btn.addEventListener('click', async (ev) => {
        ev.stopPropagation();
        if (emoteBtn.disabled) return;
        const room = roomSel.value;
        try {
          emoteBtn.disabled = true;
          emoteBtn.title = 'Sending emote...';
          const res = await api.emote(room, e.type);
          if (res.status === 401) { location.href='/login'; return; }
          if (!res.ok) {
            const txt = (await res.text()).trim();
            showMsg(`Emote failed: ${res.status} ${txt}`, 'err');
          } else {
            showMsg('Emote sent!', 'ok');
            pollState.soon();
          }
        } catch (err) {
          showMsg(`Emote error: ${err?.message||err}`, 'err');
        } finally {
          closeEmoteMenu();
          emoteBtn.disabled = false;
          emoteBtn.title = 'Send an emote';
        }
      });
      emoteMenu.appendChild(btn);
    }
  }
  function openEmoteMenu(){
    if (emoteBtn.disabled) return;
    emoteMenu.classList.add('open');
    emoteBtn.setAttribute('aria-expanded', 'true');
  }
  function closeEmoteMenu(){
    emoteMenu.classList.remove('open');
    emoteBtn.setAttribute('aria-expanded', 'false');
  }
  emoteBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    if (emoteMenu.classList.contains('open')) closeEmoteMenu();
    else openEmoteMenu();
  });
  document.addEventListener('click', (e) => {
    if (!emoteMenu.contains(e.target) && e.target !== emoteBtn) {
      closeEmoteMenu();
    }
  });
  buildEmoteMenu();

  if (showHandBtn && hideHandBtn) {
    hideHandBtn.disabled = true;
    showHandBtn.addEventListener('click', async () => {
      const room = roomSel.value;
      try {
        pulse(showHandBtn);
        showHandBtn.disabled = true;
        hideHandBtn.disabled = true;
        const res = await api.showHand(room, true);
        if (res.status === 401) { location.href = '/login'; return; }
        if (!res.ok) {
          const txt = (await res.text()).trim();
          showMsg(`Show hand failed: ${res.status} ${txt}`, 'err');
          return;
        }
        showMsg('You are now showing your hand', 'ok');
        pollState.soon();
      } catch (e) {
        showMsg(`Show hand error: ${e.message}`, 'err');
      }
    });

    hideHandBtn.addEventListener('click', async () => {
      const room = roomSel.value;
      try {
        pulse(hideHandBtn);
        showHandBtn.disabled = true;
        hideHandBtn.disabled = true;
        const res = await api.showHand(room, false);
        if (res.status === 401) { location.href = '/login'; return; }
        if (!res.ok) {
          const txt = (await res.text()).trim();
          showMsg(`Hide hand failed: ${res.status} ${txt}`, 'err');
          return;
        }
        showMsg('You are now hiding your hand', 'ok');
        pollState.soon();
      } catch (e) {
        showMsg(`Hide hand error: ${e.message}`, 'err');
      }
    });
  }

  joinBtn.addEventListener('click', async () => {
    pulse(joinBtn);
    const room = roomSel.value, stack = stackInp.value;
    try {
      statusEl.textContent='joining…'; statusEl.className='pill';
      const res = await api.join(room, stack);
      if (res.status === 401) { location.href='/login'; return; }
      if (!res.ok) {
        const errTxt = (await res.text()).trim();
        showMsg(`Join failed: ${res.status} ${errTxt}`, 'err');
        statusEl.textContent='join failed'; statusEl.className='pill err';
        return;
      }
      showMsg(`Joined room ${room}`, 'ok');
      setQueuedDisplay('');
      statusEl.textContent='joined'; statusEl.className='pill ok';
      pollState.now();
    } catch (e) { showMsg(`Join error: ${e?.message||e}`, 'err'); statusEl.textContent='join error'; statusEl.className='pill err'; }
  });

  leaveBtn.addEventListener('click', async () => {
    pulse(leaveBtn);
    const room = roomSel.value;
    try {
      const res = await api.leave(room);
      if (res.status === 401) { location.href='/login'; return; }
      if (!res.ok) { showMsg(`Leave failed: ${res.status} ${(await res.text())||''}`, 'err'); return; }
      showMsg(`Left room ${room}`, 'ok');
      renderState(null);
      setQueuedDisplay('');
      statusEl.textContent='left'; statusEl.className='pill';
    } catch (e) { showMsg(`Leave error: ${e.message}`, 'err'); }
  });

  sitInBtn.addEventListener('click', async () => {
    pulse(sitInBtn);
    const room = roomSel.value;
    try {
      const res = await api.sit(room, true);
      if (res.status === 401) { location.href='/login'; return; }
      if (!res.ok) { showMsg(`Sit in failed: ${res.status} ${(await res.text())||''}`, 'err'); return; }
      showMsg('Sat in', 'ok'); pollState.soon();
    } catch (e) { showMsg(`Sit in error: ${e.message}`, 'err'); }
  });

  sitOutBtn.addEventListener('click', async () => {
    pulse(sitOutBtn);
    const room = roomSel.value;
    try {
      const res = await api.sit(room, false);
      if (res.status === 401) { location.href='/login'; return; }
      if (!res.ok) { showMsg(`Sit out failed: ${res.status} ${(await res.text())||''}`, 'err'); return; }
      showMsg('Sat out', 'ok'); pollState.soon();
    } catch (e) { showMsg(`Sit out error: ${e.message}`, 'err'); }
  });

  function sliderToAmount(sliderVal) {
    const t = Math.min(1, Math.max(0, sliderVal / 100));
    const eased = t * t;
    return currentRaiseMin + (currentRaiseMax - currentRaiseMin) * eased;
  }
  function amountToSlider(amount) {
    if (currentRaiseMax <= currentRaiseMin) return 0;
    const norm = (amount - currentRaiseMin) / (currentRaiseMax - currentRaiseMin);
    const clamped = Math.min(1, Math.max(0, norm));
    const t = Math.sqrt(clamped);
    return t * 100;
  }

  async function doActionWithQueued(act, amt=0, displayAmt=null) {
    const room = roomSel.value;
    const btnMap = { check: checkBtn, call: callBtn, raise: raiseBtn, fold: foldBtn, clear: clearBtn };
    pulse(btnMap[act]);
    try {
      const res = await api.action(room, act, amt);
      if (res.status === 401) { location.href='/login'; return; }
      if (!res.ok) { showMsg(`Action failed: ${res.status} ${(await res.text())||''}`, 'err'); return; }

      if (act === 'raise') {
        const shown = displayAmt != null ? displayAmt : amt;
        setQueuedDisplay(`Raise to ${shown.toFixed(2)}`);
        showMsg(`Action: raise to ${shown.toFixed(2)}`, 'ok');
        safePlayChip();
      }
      else if (act === 'call') { setQueuedDisplay('Call'); showMsg('Action: call', 'ok'); safePlayChip(); }
      else if (act === 'check') { setQueuedDisplay('Check'); showMsg('Action: check', 'ok'); safePlayCheck(); }
      else if (act === 'fold') { setQueuedDisplay('Fold'); showMsg('Action: fold', 'ok'); }
      else if (act === 'clear') { setQueuedDisplay(''); showMsg('Cleared action', 'ok'); }

      pollState.soon();
    } catch (e) { showMsg(`Action error: ${e.message}`, 'err'); }
  }

  checkBtn.addEventListener('click', () => doActionWithQueued('check'));
  callBtn.addEventListener('click', () => {
    const amt = Number(callAmtEl.dataset.val)||0;
    doActionWithQueued('call', amt);
  });
  raiseBtn.addEventListener('click', () => {
    const raiseTo = Number(raiseAmt.value) || 0;
    let handBet = Number(lastHandCurrentBet) || 0;
    if (handBet === 0) {
      const minAdd = Number(lastMinRaiseAdd) || 0;
      if (currentRaiseMin > 0) {
        const inferred = currentRaiseMin - minAdd;
        if (inferred >= 0) handBet = inferred;
      }
    }
    let sendAmt = raiseTo - handBet;
    if (sendAmt < 0) sendAmt = 0;
    doActionWithQueued('raise', sendAmt, raiseTo);
  });
  foldBtn.addEventListener('click', () => doActionWithQueued('fold'));
  clearBtn.addEventListener('click', () => doActionWithQueued('clear'));
  ledgerBtn.addEventListener('click', () => { location.href = '/ledger'; });

  let rafId = 0, baseDeadlineMsGlobal = 0, skewMsGlobal = 0, tbMsGlobal = 0;

  function getTimerEls(){
    const basePill = document.querySelector('.seat.acting .base-timer');
    const baseVal  = basePill ? basePill.querySelector('.tval') : null;
    const tbPill   = basePill ? document.querySelector('.seat.acting .tb-timer') : null;
    const tbVal    = tbPill ? tbPill.querySelector('.tval') : null;
    return { basePill, baseVal, tbPill, tbVal };
  }

  function setTimerClasses(pill, msLeft){
    if (!pill) return;
    pill.classList.remove('timer-ok','timer-warn','timer-bad');
    const s = msLeft/1000;
    if (s <= 5) pill.classList.add('timer-bad');
    else if (s <= 10) pill.classList.add('timer-warn');
    else pill.classList.add('timer-ok');
  }

  function startCountdown(baseDeadlineMs, serverNowMs, timebankMs){
    baseDeadlineMsGlobal = Number(baseDeadlineMs)||0;
    tbMsGlobal = Math.max(0, Number(timebankMs)||0);
    skewMsGlobal = Number(serverNowMs||0) - Date.now();
    if (rafId) cancelAnimationFrame(rafId);

    const { basePill, baseVal } = getTimerEls();
    if (!baseDeadlineMsGlobal || !basePill || !baseVal) return;

    const tick = () => {
      const els = getTimerEls();
      if (!els.basePill || !els.baseVal) { cancelAnimationFrame(rafId); rafId=0; return; }

      const nowMs = Date.now() + skewMsGlobal;

      if (nowMs <= baseDeadlineMsGlobal) {
        const remaining = Math.max(0, baseDeadlineMsGlobal - nowMs);
        const secs = remaining / 1000;
        els.baseVal.textContent = secs >= 10 ? secs.toFixed(0)+'s' : secs.toFixed(1)+'s';
        setTimerClasses(els.basePill, remaining);
        if (els.tbPill) els.tbPill.style.display = 'none';
      } else {
        if (tbMsGlobal > 0 && els.tbPill && els.tbVal) {
          const overMs = nowMs - baseDeadlineMsGlobal;
          const leftMs = Math.max(0, tbMsGlobal - overMs);
          els.basePill.style.display = 'none';
          if (leftMs > 0) {
            els.tbPill.style.display = 'inline-flex';
            const secs = leftMs / 1000;
            els.tbVal.textContent = secs >= 10 ? secs.toFixed(0)+'s' : secs.toFixed(1)+'s';
            setTimerClasses(els.tbPill, leftMs);
          } else {
            els.tbPill.style.display = 'none';
            cancelAnimationFrame(rafId); rafId=0;
            return;
          }
        } else {
          els.baseVal.textContent = '0.0s';
          setTimerClasses(els.basePill, 0);
          cancelAnimationFrame(rafId); rafId=0;
          return;
        }
      }

      rafId = requestAnimationFrame(tick);
    };
    tick();
  }

  function stopCountdown(){ if (rafId) cancelAnimationFrame(rafId); rafId = 0; }

  const SEAT_RX = 46, SEAT_RY = 38, BET_RX=31, BET_RY=26;
  function heroPolar(){ return Math.PI/2; }
  function otherPolars(n){ if(!n) return []; const step=2*Math.PI/(n+1); return Array.from({length:n},(_,k)=>Math.PI/2+(k+1)*step); }
  function ellipseXY(theta, rx, ry){ const cx=50, cy=50; return { x: cx + rx*Math.cos(theta), y: cy + ry*Math.sin(theta) }; }

  const potEl = $('pot'), boardEl = $('board'), seatsLayer = $('seatsLayer'), betsLayer = $('betsLayer'), myHandEl = $('myHand');
  const showdownBanner = $('showdownBanner'), showdownList = $('showdownList'), actionMsgEl = $('actionMsg');

  const lastEmoteKeyByName = new Map();
  const emoteHoldUntilByName = new Map();

  function secondsLeft(nextAtMs, serverNowMs){
    if (!Number.isFinite(nextAtMs) || nextAtMs <= 0) return 0;
    const left = Math.max(0, nextAtMs - serverNowMs);
    return left / 1000;
  }
  function formatSecs(secs){
    if (secs >= 10) return `${Math.ceil(secs)}s`;
    if (secs >= 1) return `${secs.toFixed(1)}s`;
    return `${(secs*1000|0)}ms`;
  }
  function updateEmoteCooldownUI(hero, handServerNowMs){
    const nextAt = Number(hero?.nextEmoteAt ?? hero?.NextEmoteAt ?? 0);
    let nowMs = Number(handServerNowMs);
    if (!Number.isFinite(nowMs) || nowMs <= 0) nowMs = Date.now();
    const secs = secondsLeft(nextAt, nowMs);

    if (secs > 0) {
      emoteCooldownEl.textContent = `Cooldown: ${formatSecs(secs)}`;
      emoteBtn.disabled = true;
      emoteBtn.title = `Cooldown ${formatSecs(secs)}`;
      closeEmoteMenu();
      emoteMenu.querySelectorAll('.emote-item').forEach(b => b.setAttribute('disabled',''));
    } else {
      emoteCooldownEl.textContent = 'Ready';
      emoteBtn.disabled = false;
      emoteBtn.title = 'Send an emote';
      emoteMenu.querySelectorAll('.emote-item').forEach(b => b.removeAttribute('disabled'));
    }
  }

  function fmtMoney(v){ const n=Number(v); return Number.isFinite(n)?n.toFixed(2):(v??''); }

  function cardURL(card){
    const s = (card.suit||card.Suit||'').toString().toUpperCase();
    const r = (card.rank||card.Rank||'');
    const suit = s==='S'?'SPADE':s==='H'?'HEART':s==='D'?'DIAMOND':s==='C'?'CLUB':'';
    let rt = ''; const n=Number(r);
    if(Number.isFinite(n)){
      rt = (n>=2&&n<=10)?String(n): n===11?'11-JACK':n===12?'12-QUEEN':n===13?'13-KING':(n===14||n===1)?'1-ACE':'';
    } else {
      const up=String(r).toUpperCase();
      rt = up==='A'?'1-ACE':up==='K'?'13-KING':up==='Q'?'12-QUEEN':up==='J'?'11-JACK':'';
    }
    return (suit && rt) ? `/static/card_svgs/${suit}-${rt}.svg` : null;
  }

  function renderCardImg(card) {
    const img = document.createElement('img');
    img.className = 'cardimg';
    const url = cardURL(card);
    if (url) {
      img.src = url;
      img.alt = url.split('/').pop() || 'card';
    } else {
      img.src = '/static/card_svgs/card_back.png';
      img.alt = 'card';
    }
    return img;
  }

  function getShowdownHandsArray(hand) {
    const raw = hand?.showDownHands ?? hand?.ShowDownHands ?? null;
    return Array.isArray(raw) ? raw : [];
  }

  const pollState = (() => {
    let timer=null;
    async function tick(){
      const room = roomSel.value;
      statusEl.textContent='updating…'; statusEl.className='pill';
      try {
        const res = await api.state(room);
        if (!res.ok) {
          const txt = (await res.text()).trim().toLowerCase();
          if (res.status === 400 && txt.includes('player not in room')) {
            renderState(null);
            showMsg('You are not in the room', 'err');
            statusEl.textContent='not in room'; statusEl.className='pill err';
          } else if (res.status === 401) {
            location.href='/login'; return;
          } else {
            statusEl.textContent='error'; statusEl.className='pill err';
          }
          return;
        }
        const st = JSON.parse(await res.text());
        renderState(st);
        statusEl.textContent='connected'; statusEl.className='pill';
      } catch (e) {
        statusEl.textContent='error'; statusEl.className='pill err';
      } finally { timer=setTimeout(tick, 1000); }
    }
    function start(){ if(!timer) timer=setTimeout(tick, 80); }
    function stop(){ if(timer){ clearTimeout(timer); timer=null; } }
    function now(){ stop(); start(); }
    function soon(){ stop(); timer=setTimeout(tick, 300); }
    document.addEventListener('visibilitychange', ()=>{ if(document.hidden) stop(); else now(); });
    start();
    return { start, stop, now, soon };
  })();

  let prevActionMessage = '';

  function renderState(st){
    if (!st || (Object.keys(st).length===0)){
      potEl.textContent='—';
      stopCountdown();
      boardEl.innerHTML=''; seatsLayer.innerHTML=''; betsLayer.innerHTML=''; myHandEl.innerHTML='';
      showdownBanner.style.display='none'; showdownList.style.display='none'; showdownList.innerHTML='';
      actionMsgEl.style.display='none';
      serverMsgEl.className = 'server-message';
      lastHeroWas67 = false;
      lastBoardLen = 0;
      if (timebankDisplayEl) timebankDisplayEl.textContent = '';
      updateEmoteCooldownUI(null, Date.now());
      if (showHandBtn && hideHandBtn) {
        showHandBtn.disabled = false;
        hideHandBtn.disabled = true;
      }
      return;
    }

    const hand = st.hand || st.Hand || {};
    let serverNowMs = Number(hand.serverCurrentTimeUnixMs ?? hand.ServerCurrentTimeUnixMs);
    if (!Number.isFinite(serverNowMs) || serverNowMs <= 0) serverNowMs = Date.now();

    const actorName = String(hand.actionPlayerName ?? hand.ActionPlayerName ?? '').trim();
    const sbName = String(hand.smallBlindName   ?? hand.SmallBlindName   ?? '').trim();
    const bbName = String(hand.bigBlindName     ?? hand.BigBlindName     ?? '').trim();
    const dealerName = String(hand.dealerName    ?? hand.DealerName    ?? '').trim();

    potEl.textContent = fmtMoney(hand.pot ?? hand.Pot ?? '');

    const board = Array.isArray(hand.board ?? hand.Board) ? (hand.board ?? hand.Board) : [];
    const currentBoardLen = board.length;
    boardEl.innerHTML='';
    board.forEach(c=>boardEl.appendChild(renderCardImg(c)));
    if (currentBoardLen > lastBoardLen) {
      for (let i = lastBoardLen; i < currentBoardLen; i++) {
        setTimeout(() => safePlayDeal(), (i - lastBoardLen) * 120);
      }
    }
    lastBoardLen = currentBoardLen;

    const rawPlayers = Array.isArray(st.players || st.Players)
      ? (st.players || st.Players).slice()
      : [];

    const heroIdFromState = st.heroID || st.heroId || st.HeroID || null;
    let heroIdx = -1;

    if (heroIdFromState) {
      const heroIdNorm = String(heroIdFromState).trim();
      heroIdx = rawPlayers.findIndex(p =>
        String(p.id ?? p.ID ?? '').trim() === heroIdNorm
      );
    }

    if (heroIdx < 0 && myName) {
      const myLower = myName.trim().toLowerCase();
      heroIdx = rawPlayers.findIndex(p =>
        String(p.name ?? p.Name ?? '').trim().toLowerCase() === myLower
      );
    }

    if (heroIdx < 0) {
      heroIdx = rawPlayers.findIndex(p => {
        const h = p.hand ?? p.Hand;
        return Array.isArray(h) && h.length > 0;
      });
    }

    if (heroIdx < 0) heroIdx = 0;

    const hero = rawPlayers[heroIdx] || null;
    const others = rawPlayers.length
      ? [...rawPlayers.slice(heroIdx+1), ...rawPlayers.slice(0, heroIdx)]
      : [];

    const heroTheta = heroPolar();
    const othersTheta = otherPolars(others.length);
    const players = hero ? [hero, ...others] : others.slice();
    const angles  = hero ? [heroTheta, ...othersTheta] : othersTheta.slice();

    if (showHandBtn && hideHandBtn) {
      const heroShowing = !!(hero && (hero.showingHand ?? hero.ShowingHand));
      showHandBtn.disabled = heroShowing;
      hideHandBtn.disabled = !heroShowing;
    }

    if (timebankDisplayEl) {
      if (hero) {
        const tbMs = Number(hero.timeBankUnixMs ?? hero.TimeBankUnixMs ?? 0) || 0;
        const label = tbMs > 0 ? (tbMs/1000).toFixed(1) + 's' : '0.0s';
        timebankDisplayEl.textContent = 'Timebank: ' + label;
      } else {
        timebankDisplayEl.textContent = '';
      }
    }

    seatsLayer.innerHTML='';
    players.forEach((p,i)=>{
      const pos = ellipseXY(angles[i], SEAT_RX, SEAT_RY);
      const name = String(p.name ?? p.Name ?? '').trim() || '(player)';
      const isActing = !!(actorName && name === actorName);
      const isSB = (name === sbName);
      const isBB = (name === bbName);
      const isDealer = (name === dealerName);
      const tbMs = Number(p.timeBankUnixMs ?? p.TimeBankUnixMs ?? 0) || 0;
      const isHeroSeat = (i === 0 && !!hero);

      const seat=document.createElement('div');
      seat.className='seat'+(isHeroSeat?' hero':'')+(isActing?' acting':'')+((p.sittingOut ?? p.SittingOut)?' sittingout':'')+((p.folded ?? p.Folded)?' folded':'');
      seat.style.left=pos.x+'%'; seat.style.top=pos.y+'%';

      const nameWrap = document.createElement('div');
      const nameSpan = document.createElement('strong'); nameSpan.textContent = name;
      nameWrap.appendChild(nameSpan);

      if (isSB) {
        const sbSpan = document.createElement('span');
        sbSpan.className = 'sb-indicator';
        sbSpan.textContent = 'SB';
        nameWrap.appendChild(sbSpan);
      }
      if (isBB) {
        const bbSpan = document.createElement('span');
        bbSpan.className = 'bb-indicator';
        bbSpan.textContent = 'BB';
        nameWrap.appendChild(bbSpan);
      }
      if (isDealer) {
        const dealerSpan = document.createElement('span');
        dealerSpan.className = 'dealer-indicator';
        dealerSpan.textContent = 'D';
        nameWrap.appendChild(dealerSpan);
      }

      if (isActing) {
        const baseTimer = document.createElement('span');
        baseTimer.className = 'act-timer-seat timer-ok base-timer';
        const clock = document.createTextNode('⏱ ');
        const tval = document.createElement('span');
        tval.className = 'tval';
        tval.textContent = '0.0s';
        baseTimer.appendChild(clock); baseTimer.appendChild(tval);
        nameWrap.appendChild(baseTimer);

        if (tbMs > 0) {
          const tbTimer = document.createElement('span');
          tbTimer.className = 'act-timer-seat timer-ok tb-timer';
          tbTimer.style.display = 'none';
          const icon = document.createTextNode('⏳ ');
          const tbVal = document.createElement('span');
          tbVal.className = 'tval';
          tbVal.textContent = (tbMs/1000).toFixed(1)+'s';
          tbTimer.appendChild(icon);
          tbTimer.appendChild(tbVal);
          nameWrap.appendChild(tbTimer);
        }
      }

      const chips=document.createElement('div'); chips.className='chips';
      const chipImg=document.createElement('img'); chipImg.src='/static/chips/chips.png'; chipImg.alt='';
      const chipsText=document.createElement('span'); chipsText.textContent=fmtMoney(p.stack ?? p.Stack ?? '');
      chips.appendChild(chipImg); chips.appendChild(chipsText);

      seat.appendChild(nameWrap);
      seat.appendChild(chips);

      const sittingOut = !!(p.sittingOut ?? p.SittingOut);
      const folded = !!(p.folded ?? p.Folded);
      const handArr = Array.isArray(p.hand ?? p.Hand) ? (p.hand ?? p.Hand) : [];
      const showing = !!(p.showingHand ?? p.ShowingHand);

      if (!sittingOut && !folded) {
        const holes = document.createElement('div');
        holes.className = 'seat-holes';

        if (showing && handArr.length) {
          handArr.slice(0,2).forEach((c) => {
            const img = document.createElement('img');
            const url = cardURL(c);
            img.src = url || '/static/card_svgs/card_back.png';
            img.alt = url ? url.split('/').pop() : 'card';
            holes.appendChild(img);
          });
        } else {
          const card1 = document.createElement('img');
          card1.src = '/static/card_svgs/card_back.png';
          card1.alt = 'card back';
          const card2 = document.createElement('img');
          card2.src = '/static/card_svgs/card_back.png';
          card2.alt = 'card back';
          holes.appendChild(card1);
          holes.appendChild(card2);
        }

        seat.appendChild(holes);
      }

      const emTextRaw = String(p.emoteText ?? p.EmoteText ?? '').trim();
      const emText    = emTextRaw;
      const emAudio   = String(p.emoteAudio ?? p.EmoteAudio ?? '').trim();
      const emEnds    = Number(p.emoteEndsUnixMs ?? p.EmoteEndsUnixMs ?? 0) || 0;

      const holdKey = name;
      const holdUntil = emoteHoldUntilByName.get(holdKey) || 0;
      const shouldShow = emText && (serverNowMs <= Math.max(emEnds, holdUntil));

      if (emText && emEnds > 0) {
        if (serverNowMs <= emEnds + 200) {
          emoteHoldUntilByName.set(holdKey, emEnds + 200);
        }
      }

      const emKey = emText && emEnds ? `${holdKey}|${emEnds}|${emText}` : '';
      const lastKey = lastEmoteKeyByName.get(holdKey);
      if (emKey && emKey !== lastKey) {
        playEmoteAudio(emAudio);
        lastEmoteKeyByName.set(holdKey, emKey);
      }

      let bubble = seat.querySelector('.emote-bubble');
      if (shouldShow) {
        if (!bubble) {
          bubble = document.createElement('div');
          bubble.className = 'emote-bubble sticky';
          seat.appendChild(bubble);
        }
        bubble.textContent = emText;
      } else if (bubble) {
        bubble.remove();
      }

      seatsLayer.appendChild(seat);
    });

    renderBets(players, angles);

    myHandEl.innerHTML='';
    const myCards = players[0] ? (players[0].hand ?? players[0].Hand ?? []) : [];
    if (Array.isArray(myCards) && myCards.length) {
      myCards.forEach(c=>myHandEl.appendChild(renderCardImg(c)));
    }

    if (Array.isArray(myCards) && myCards.length === 2) {
      const r1 = parseInt(myCards[0].rank ?? myCards[0].Rank ?? 0, 10);
      const r2 = parseInt(myCards[1].rank ?? myCards[1].Rank ?? 0, 10);
      const is67 = ((r1 === 6 && r2 === 7) || (r1 === 7 && r2 === 6));
      if (is67 && !lastHeroWas67) { safePlay67(); lastHeroWas67 = true; }
      else if (!is67) { lastHeroWas67 = false; }
    } else { lastHeroWas67 = false; }

    const actionMsg = String(hand.currentActionMessage ?? hand.CurrentActionMessage ?? '').trim();
    if (actionMsg){
      actionMsgEl.textContent=actionMsg;
      actionMsgEl.style.display='block';
      if (actionMsg !== prevActionMessage) {
        if (/\b(check|checked)\b/i.test(actionMsg)) {
          safePlayCheck();
        } else if (/\b(call|called|calls|raise|raised|raises|re-?raise|reraise|bet|bets)\b/i.test(actionMsg)) {
          safePlayChip();
          if (!/^\s*fold/i.test(currentQueued)) {
            setQueuedDisplay('');
            showMsg('Your queued action was cleared (opponent raised)', 'ok');
          }
        }
      }
    } else { actionMsgEl.style.display='none'; actionMsgEl.textContent=''; }
    prevActionMessage = actionMsg;

    const deadlineMs = Number(hand.actionDeadlineUnixMs ?? hand.ActionDeadlineUnixMs ?? 0) || 0;
    let actorTimebankMs = 0;
    if (actorName) {
      const actor = players.find(p => (String(p.name ?? p.Name ?? '').trim() === actorName));
      if (actor) {
        actorTimebankMs = Number(actor.timeBankUnixMs ?? actor.TimeBankUnixMs ?? 0) || 0;
      }
    }
    if (deadlineMs > 0) startCountdown(deadlineMs, serverNowMs, actorTimebankMs);
    else stopCountdown();

    const sdMsg = String(hand.showDownMessage ?? hand.ShowDownMessage ?? '').trim();
    const sdHands = getShowdownHandsArray(hand);
    const street = String(hand.street ?? hand.Street ?? '').toLowerCase();
    const hasShowdown = (street==='showdown') && (sdMsg.length > 0 || (Array.isArray(sdHands) && sdHands.length > 0));
    showdownBanner.style.display = hasShowdown ? 'block' : 'none';
    showdownBanner.textContent = hasShowdown ? (sdMsg || 'Showdown') : '';
    showdownList.innerHTML = '';
    if (hasShowdown && Array.isArray(sdHands) && sdHands.length) {
      const nameById = new Map();
      (st.players || st.Players || []).forEach(p => {
        const pid = String(p.id ?? p.ID ?? p.playerId ?? p.PlayerID ?? '').trim();
        const nm  = String(p.name ?? p.Name ?? '').trim();
        if (pid) nameById.set(pid, nm);
        if (nm && !nameById.has(nm)) nameById.set(nm, nm);
      });
      sdHands.forEach(entry => {
        const cards = entry.Hand ?? entry.hand ?? null;
        if (!Array.isArray(cards) || cards.length === 0) return;
        const pidOrName = String(entry.PlayerName ?? entry.playerName ?? entry.PlayerID ?? entry.playerId ?? '').trim();
        const displayName = nameById.get(pidOrName) || pidOrName || 'player';
        const row = document.createElement('div');
        row.className = 'entry';
        const who = document.createElement('div'); who.className = 'who'; who.textContent = displayName;
        const cardsWrap = document.createElement('div'); cardsWrap.className = 'cards';
        cards.forEach(c => cardsWrap.appendChild(renderCardImg(c)));
        row.appendChild(who); row.appendChild(cardsWrap);
        showdownList.appendChild(row);
      });
      showdownList.style.display = showdownList.children.length ? 'flex' : 'none';
    } else {
      showdownList.style.display = 'none';
    }

    const avail = new Set(hand.avaliableActions || hand.availableActions || []);
    const myBet   = Number((players[0] && (players[0].currentBet ?? players[0].CurrentBet)) ?? 0) || 0;
    const tableBet = Number(hand.currentBet ?? hand.CurrentBet ?? 0) || 0;
    lastHandCurrentBet = tableBet;

    const callAmt = Math.max(0, tableBet - myBet);
    callAmtEl.textContent = fmtMoney(callAmt);
    callAmtEl.dataset.val = String(callAmt);

    const ONE_BB = 0.25;
    let minAdd = Number(hand.raiseAmount ?? hand.RaiseAmount ?? 0);
    if (!Number.isFinite(minAdd) || minAdd <= 0) {
      if (tableBet <= myBet) { minAdd = ONE_BB; }
      else { minAdd = Math.max(ONE_BB, tableBet - myBet); }
    }
    lastMinRaiseAdd = minAdd;

    const myStack = Number((players[0] && (players[0].stack ?? players[0].Stack)) ?? 0) || 0;
    const baseRaiseTo = tableBet + minAdd;
    const maxRaiseTo = myBet + myStack;

    const sliderMin = Math.max(0, Math.round(baseRaiseTo * 100) / 100);
    const sliderMax = Math.max(sliderMin, Math.round(maxRaiseTo * 100) / 100);

    currentRaiseMin = sliderMin;
    currentRaiseMax = sliderMax;

    const ctx = [
      String(street||'').toLowerCase(),
      Number(tableBet||0).toFixed(2),
      Number(myBet||0).toFixed(2),
      Number(minAdd||0).toFixed(2),
      String(actorName||'')
    ].join('|');

    const currentInputVal = Number(raiseAmt.value || NaN);
    const outOfBounds = !Number.isFinite(currentInputVal) ||
                        currentInputVal < sliderMin - 1e-9 ||
                        currentInputVal > sliderMax + 1e-9;

    if (ctx !== window._lastSliderCtx || outOfBounds) {
      raiseAmt.value = sliderMin.toFixed(2);
      raiseAmtLabel.textContent = sliderMin.toFixed(2);
      raiseSlider.value = amountToSlider(sliderMin);
    }
    window._lastSliderCtx = ctx;

    if (avail.has('check')) { foldBtn.disabled = true; } else { foldBtn.disabled = false; }
    checkBtn.disabled = !avail.has('check');
    callBtn.disabled  = !avail.has('call') || callAmt <= 0.0001;
    raiseBtn.disabled = !avail.has('raise');

    updateEmoteCooldownUI(players[0] || null, serverNowMs);
  }

  function renderBets(players, angles){
    betsLayer.innerHTML='';
    players.forEach((p,i)=>{
      const bet=Number(p.currentBet ?? p.CurrentBet ?? 0);
      if(!Number.isFinite(bet) || bet<=0) return;
      const pos=ellipseXY(angles[i], BET_RX, BET_RY);
      const wrap=document.createElement('div'); wrap.className='bet-chip'; wrap.style.left=pos.x+'%'; wrap.style.top=pos.y+'%';
      const img=document.createElement('img'); img.src='/static/chips/chips.png'; img.alt='bet';
      const amt=document.createElement('div'); amt.className='amt'; amt.textContent=fmtMoney(bet);
      wrap.appendChild(img); wrap.appendChild(amt); betsLayer.appendChild(wrap);
    });
  }

  function syncFromSlider(){
    const sliderVal = Number(raiseSlider.value)||0;
    const amt = sliderToAmount(sliderVal);
    raiseAmt.value = amt.toFixed(2);
    raiseAmtLabel.textContent = amt.toFixed(2);
  }
  function syncFromBox(){
    const v=Number(raiseAmt.value)||0;
    const clamped = Math.max(currentRaiseMin, Math.min(currentRaiseMax, v));
    raiseAmt.value = clamped.toFixed(2);
    raiseAmtLabel.textContent = clamped.toFixed(2);
    raiseSlider.value = amountToSlider(clamped);
  }
  raiseSlider.addEventListener('input', syncFromSlider);
  raiseAmt.addEventListener('input',    syncFromBox);

  if (raiseUp && raiseDown) {
    const STEP = 0.25;

    raiseUp.addEventListener('click', () => {
      let val = Number(raiseAmt.value);
      if (!Number.isFinite(val) || val < currentRaiseMin) val = currentRaiseMin;
      val += STEP;
      if (val > currentRaiseMax) val = currentRaiseMax;
      raiseAmt.value = val.toFixed(2);
      raiseAmtLabel.textContent = val.toFixed(2);
      raiseSlider.value = amountToSlider(val);
    });

    raiseDown.addEventListener('click', () => {
      let val = Number(raiseAmt.value);
      if (!Number.isFinite(val) || val < currentRaiseMin) val = currentRaiseMin;
      val -= STEP;
      if (val < currentRaiseMin) val = currentRaiseMin;
      raiseAmt.value = val.toFixed(2);
      raiseAmtLabel.textContent = val.toFixed(2);
      raiseSlider.value = amountToSlider(val);
    });
  }

  (async () => {
    try {
      const res = await api.state(roomSel.value);
      if (res.ok) renderState(JSON.parse(await res.text()));
    } catch {}
  })();
});
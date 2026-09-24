package docs

// pageStyle is the stylesheet every generated page carries, inlined because each file under
// site/ has to render without fetching anything (docs/site.md). The palette is the website's;
// the two are separate copies on purpose, since sharing one would mean the page fetches a
// stylesheet and breaks the promise it makes about itself.
const pageStyle = baseStyle + docStyle

const baseStyle = `:root{
  --bg:#ffffff; --surface:#f5f7f8; --line:#e2e6e9; --line-strong:#cdd3d8;
  --fg:#0f1417; --dim:#4f5a64; --faint:#66717c;
  --signal:#0f766e; --signal-strong:#0b5f58; --signal-soft:#e7f4f2;
  --cell:#f5f7f8; --rule:#e2e6e9; --rule-soft:#edf0f2;
  --mono:ui-monospace,"SF Mono",SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;
  --sans:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;
  --gut:clamp(1rem,4vw,2.5rem);
}
@media (prefers-color-scheme:dark){
  :root{
    --bg:#0c1013; --surface:#11171b; --line:#232e35; --line-strong:#33424c;
    --fg:#edf1f3; --dim:#a3aeb7; --faint:#8b97a1;
    --signal:#2dd4bf; --signal-strong:#5eead4; --signal-soft:#0e2826;
    --cell:#11171b; --rule:#232e35; --rule-soft:#1a2328;
  }
}
*,*::before,*::after{box-sizing:border-box}
html{-webkit-text-size-adjust:100%; scroll-behavior:smooth; color-scheme:light dark}
@media (prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
body{margin:0; background:var(--bg); color:var(--fg); font-family:var(--sans);
  font-size:clamp(1rem,.96rem + .2vw,1.0625rem); line-height:1.65; -webkit-font-smoothing:antialiased}
a{color:var(--signal); text-underline-offset:.2em}
a:hover{color:var(--signal-strong)}
:focus-visible{outline:2px solid var(--signal); outline-offset:3px; border-radius:4px}
.wrap{max-width:84rem; margin:0 auto; padding:0 var(--gut)}
.top{display:flex; align-items:center; justify-content:space-between; gap:.5rem 1.5rem; flex-wrap:wrap;
  min-height:3.75rem; border-bottom:1px solid var(--line)}
.brand{display:inline-flex; align-items:center; gap:.55rem; min-height:44px; font-weight:700; font-size:1.1rem;
  letter-spacing:-.02em; text-decoration:none; color:var(--fg)}
.brand svg{width:26px; height:26px; flex:none}
.topnav{display:flex; flex-wrap:wrap; gap:0 .25rem; font-size:.95rem}
.topnav a{display:inline-flex; align-items:center; min-height:44px; padding:0 .7rem; border-radius:8px;
  text-decoration:none; color:var(--dim); font-weight:500}
.topnav a:hover{color:var(--fg); background:var(--surface)}
.topnav a.menu{display:none}
footer{padding:2.5rem 0 3.5rem; margin-top:3rem; border-top:1px solid var(--line); color:var(--faint); font-size:.88rem}
footer p{max-width:56rem; margin:0}
`

const docStyle = `.layout{display:grid; grid-template-columns:16rem minmax(0,1fr); gap:3rem; align-items:start}
.side{position:sticky; top:0; padding:2rem 0; max-height:100vh; overflow-y:auto}
@media (max-width:60rem){
  .layout{grid-template-columns:minmax(0,1fr); gap:0}
  .side{position:static; order:2; max-height:none; padding:1.5rem 0 0; border-top:1px solid var(--line)}
  .topnav a.menu{display:inline-flex}
}
@media (max-width:34rem){.topnav a.opt{display:none} .topnav a{padding:0 .5rem}}
.sidehead{margin:1.5rem 0 .4rem; font:700 .72rem/1 var(--sans); letter-spacing:.08em;
  text-transform:uppercase; color:var(--faint)}
.sidehead:first-child{margin-top:0}
.side ul{list-style:none; margin:0; padding:0}
.side a{display:flex; align-items:center; min-height:40px; text-decoration:none; color:var(--dim);
  font-size:.93rem; line-height:1.35; padding:.35rem .75rem; border-radius:8px}
.side a:hover{color:var(--fg); background:var(--surface)}
.side a.on{color:var(--signal); background:var(--signal-soft); font-weight:600}
.doc{padding:2rem 0 1rem; max-width:48rem; min-width:0}
.doc h1{font-size:clamp(1.8rem,1.3rem + 2vw,2.5rem); line-height:1.1; letter-spacing:-.03em; margin:0 0 1rem}
.doc h2{font-size:clamp(1.3rem,1.15rem + .6vw,1.55rem); letter-spacing:-.015em; line-height:1.25;
  margin:2.75rem 0 .75rem; padding-top:1.5rem; border-top:1px solid var(--line)}
.doc h3{font-size:1.12rem; letter-spacing:-.01em; margin:2rem 0 .5rem}
.doc h4{font-size:1rem; margin:1.5rem 0 .4rem; color:var(--dim)}
.doc h2 a,.doc h3 a,.doc h4 a{text-decoration:none; color:inherit}
.doc p,.doc li{max-width:42rem}
.doc p{margin:0 0 1rem}
.doc .lede{color:var(--dim); font-size:1.08rem}
.doc ul,.doc ol{margin:0 0 1rem; padding-left:1.35rem}
.doc li{margin:0 0 .45rem}
.doc li>ul,.doc li>ol{margin-top:.45rem}
.doc blockquote{margin:0 0 1rem; padding:.2rem 0 .2rem 1rem; border-left:3px solid var(--signal);
  color:var(--dim)}
.doc hr{border:0; border-top:1px solid var(--line); margin:2.5rem 0}
.doc strong{font-weight:650}
.doc img{max-width:100%}
code{font-family:var(--mono); font-size:.875em}
:not(pre)>code{background:var(--cell); border:1px solid var(--line); border-radius:5px;
  padding:.08em .35em; overflow-wrap:anywhere}
pre{background:#0e1316; color:#e6ecef; border:1px solid #222c32; border-radius:10px;
  padding:1rem 1.1rem; overflow-x:auto; margin:0 0 1.25rem; line-height:1.55}
pre code{background:none; border:0; padding:0; font-size:.84rem; color:inherit}
.scroll{overflow-x:auto; -webkit-overflow-scrolling:touch; margin:0 0 1.25rem}
table{border-collapse:collapse; width:100%; font-size:.9rem}
.doc table{display:block; overflow-x:auto; margin:0 0 1.25rem}
th,td{text-align:left; vertical-align:top; padding:.6rem .8rem .6rem 0; border-bottom:1px solid var(--line)}
th{font-size:.75rem; text-transform:uppercase; letter-spacing:.06em; color:var(--faint); font-weight:650; white-space:nowrap}
td.m{white-space:nowrap; color:var(--signal); font-family:var(--mono); font-size:.84rem}
td.dim{color:var(--dim)}
section{scroll-margin-top:1rem}
section h2{font-size:clamp(1.3rem,1.15rem + .6vw,1.55rem); margin:2.75rem 0 .35rem; padding-top:1.5rem;
  border-top:1px solid var(--line); letter-spacing:-.015em}
.note{color:var(--dim); font-size:.93rem; max-width:48rem; margin:0 0 1.25rem}
h3.sub{font-size:1rem; margin:1.75rem 0 .5rem; font-weight:650}
ul.gaps{margin:.15rem 0 0; padding-left:1.1rem; color:var(--dim)}
.cmd{padding:1rem 0; border-bottom:1px solid var(--line)}
.cmd h3{font-family:var(--mono); font-size:.9rem; margin:0 0 .25rem; color:var(--signal); font-weight:600}
.cmd p{margin:0; color:var(--dim); font-size:.93rem; max-width:44rem}
.cmd table{margin-top:.6rem}
`

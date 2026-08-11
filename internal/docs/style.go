package docs

// pageStyle is the stylesheet every generated page carries, inlined because each file under
// site/ has to render without fetching anything (docs/site.md). The palette is the website's;
// the two are separate copies on purpose, since sharing one would mean the page fetches a
// stylesheet and breaks the promise it makes about itself.
const pageStyle = baseStyle + docStyle

const baseStyle = `:root{
  --ink:#14120F; --paper:#EFEAE1;
  --teal:#3E8B79; --teal-lift:#4F9E8B;
  --bg:var(--paper); --fg:#1A1713; --dim:#6B6459; --faint:#8D857A;
  --rule:rgba(20,18,15,.14); --rule-soft:rgba(20,18,15,.07);
  --cell:rgba(20,18,15,.025); --signal:var(--teal);
  --mono:ui-monospace,"SF Mono",SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;
  --sans:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  --gut:clamp(1.25rem,4vw,2.5rem);
}
@media (prefers-color-scheme:dark){
  :root:not([data-theme="light"]){
    --bg:var(--ink); --fg:#EDE7DC; --dim:#9C9486; --faint:#787064;
    --rule:rgba(237,231,220,.16); --rule-soft:rgba(237,231,220,.07);
    --cell:rgba(237,231,220,.03); --signal:var(--teal-lift);
  }
}
:root[data-theme="dark"]{
  --bg:var(--ink); --fg:#EDE7DC; --dim:#9C9486; --faint:#787064;
  --rule:rgba(237,231,220,.16); --rule-soft:rgba(237,231,220,.07);
  --cell:rgba(237,231,220,.03); --signal:var(--teal-lift);
}
*,*::before,*::after{box-sizing:border-box}
html{-webkit-text-size-adjust:100%; scroll-behavior:smooth}
@media (prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
body{margin:0; background:var(--bg); color:var(--fg); font-family:var(--sans);
  font-size:15.5px; line-height:1.65; -webkit-font-smoothing:antialiased}
a{color:var(--signal)}
.wrap{max-width:84rem; margin:0 auto; padding:0 var(--gut)}
.top{display:flex; align-items:baseline; gap:1.5rem; flex-wrap:wrap;
  padding:1.15rem 0; border-bottom:1px solid var(--rule)}
.brand{font:600 1rem/1 var(--mono); letter-spacing:-.02em; text-decoration:none; color:var(--fg)}
.topnav{display:flex; gap:1.1rem; flex-wrap:wrap; font-family:var(--mono); font-size:12px}
.topnav a{text-decoration:none}
footer{padding:2.5rem 0 4rem; margin-top:3rem; border-top:1px solid var(--rule-soft);
  color:var(--faint); font-size:13px}
footer p{max-width:56rem; margin:0}
`

const docStyle = `.layout{display:grid; grid-template-columns:15rem minmax(0,1fr); gap:3rem; align-items:start}
@media (max-width:60rem){.layout{grid-template-columns:minmax(0,1fr); gap:1.5rem}
  .side{position:static; border-bottom:1px solid var(--rule-soft); padding-bottom:1rem}}
.side{position:sticky; top:0; padding:2rem 0; max-height:100vh; overflow-y:auto}
.sidehead{margin:1.4rem 0 .45rem; font:500 .625rem/1 var(--mono); letter-spacing:.14em;
  text-transform:uppercase; color:var(--faint)}
.sidehead:first-child{margin-top:0}
.side ul{list-style:none; margin:0; padding:0}
.side li{margin:0 0 .3rem}
.side a{display:block; text-decoration:none; color:var(--dim); font-size:13.5px; line-height:1.4;
  padding:.15rem 0 .15rem .6rem; border-left:2px solid transparent}
.side a:hover{color:var(--fg)}
.side a.on{color:var(--signal); border-left-color:var(--signal)}
.doc{padding:2rem 0 1rem; max-width:52rem; min-width:0}
.doc h1{font-size:clamp(1.6rem,4vw,2.15rem); line-height:1.15; letter-spacing:-.015em; margin:0 0 1rem}
.doc h2{font-size:1.3rem; letter-spacing:-.01em; margin:2.75rem 0 .75rem;
  padding-top:1.25rem; border-top:1px solid var(--rule-soft)}
.doc h3{font-size:1.05rem; margin:2rem 0 .5rem}
.doc h4{font-size:.95rem; margin:1.5rem 0 .4rem; color:var(--dim)}
.doc h2 a,.doc h3 a,.doc h4 a{text-decoration:none}
.doc p,.doc li{max-width:44rem}
.doc p{margin:0 0 1rem}
.doc .lede{color:var(--dim); font-size:1.02rem}
.doc ul,.doc ol{margin:0 0 1rem; padding-left:1.35rem}
.doc li{margin:0 0 .45rem}
.doc li>ul,.doc li>ol{margin-top:.45rem}
.doc blockquote{margin:0 0 1rem; padding:.1rem 0 .1rem 1rem; border-left:2px solid var(--rule);
  color:var(--dim)}
.doc hr{border:0; border-top:1px solid var(--rule-soft); margin:2.5rem 0}
.doc strong{font-weight:600}
.doc img{max-width:100%}
code{font-family:var(--mono); font-size:.875em}
:not(pre)>code{background:var(--cell); border:1px solid var(--rule-soft); border-radius:3px;
  padding:.08em .32em; word-break:break-word}
pre{background:var(--cell); border:1px solid var(--rule-soft); border-radius:4px;
  padding:.9rem 1rem; overflow-x:auto; margin:0 0 1.25rem; line-height:1.55}
pre code{background:none; border:0; padding:0; font-size:12.5px}
.scroll{overflow-x:auto; -webkit-overflow-scrolling:touch; margin:0 0 1.25rem}
table{border-collapse:collapse; width:100%; font-size:13px}
.doc table{display:block; overflow-x:auto; margin:0 0 1.25rem}
th,td{text-align:left; vertical-align:top; padding:.5rem .75rem .5rem 0;
  border-bottom:1px solid var(--rule-soft)}
th{font-family:var(--mono); font-size:11px; text-transform:uppercase; letter-spacing:.06em;
  color:var(--faint); font-weight:500; white-space:nowrap}
tbody tr:nth-child(odd){background:var(--cell)}
td.m{white-space:nowrap; color:var(--signal); font-family:var(--mono); font-size:12.5px}
td.dim{color:var(--dim)}
section{scroll-margin-top:1rem}
section h2{font-size:1.3rem; margin:2.75rem 0 .35rem; padding-top:1.25rem;
  border-top:1px solid var(--rule-soft); letter-spacing:-.01em}
.note{color:var(--dim); font-size:13.5px; max-width:52rem; margin:0 0 1.25rem}
h3.sub{font-size:.95rem; margin:1.75rem 0 .5rem; font-weight:600}
ul.gaps{margin:.15rem 0 0; padding-left:1.1rem; color:var(--dim)}
.cmd{padding:1rem 0; border-bottom:1px solid var(--rule-soft)}
.cmd h3{font-family:var(--mono); font-size:13px; margin:0 0 .2rem; color:var(--signal); font-weight:500}
.cmd p{margin:0; color:var(--dim); font-size:13.5px; max-width:44rem}
.cmd table{margin-top:.6rem}
`

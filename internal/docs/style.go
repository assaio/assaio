package docs

// pageStyle is the reference page's stylesheet, inlined because every file under site/ has to
// render without fetching anything (docs/site.md). The palette is the website's; the two are
// separate copies on purpose, since sharing one would mean the page fetches a stylesheet and
// breaks the promise it makes about itself.
const pageStyle = `:root{
  --ink:#14120F; --paper:#EFEAE1;
  --teal:#3E8B79; --teal-lift:#4F9E8B;
  --bg:var(--paper); --fg:#1A1713; --dim:#6B6459; --faint:#8D857A;
  --rule:rgba(20,18,15,.14); --rule-soft:rgba(20,18,15,.07);
  --cell:rgba(20,18,15,.025); --signal:var(--teal);
  --mono:ui-monospace,"SF Mono",SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace;
  --sans:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  --gut:clamp(1.25rem,4vw,2.5rem); --maxw:78rem;
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
  font-size:15px; line-height:1.6; -webkit-font-smoothing:antialiased}
.wrap{max-width:var(--maxw); margin:0 auto; padding:0 var(--gut)}
a{color:var(--signal)}
header{border-bottom:1px solid var(--rule); padding:3rem 0 2rem}
h1{font-size:clamp(1.6rem,4vw,2.2rem); margin:0 0 .5rem; letter-spacing:-.01em}
.lede{color:var(--dim); max-width:44rem; margin:0 0 1.25rem}
nav{display:flex; flex-wrap:wrap; gap:.35rem 1.25rem; font-family:var(--mono); font-size:12px}
nav a{text-decoration:none}
section{padding:2.5rem 0; border-bottom:1px solid var(--rule-soft)}
h2{font-size:1.15rem; margin:0 0 .35rem; letter-spacing:-.01em}
.note{color:var(--dim); font-size:13.5px; max-width:52rem; margin:0 0 1.25rem}
.scroll{overflow-x:auto; -webkit-overflow-scrolling:touch}
table{border-collapse:collapse; width:100%; font-size:13px; min-width:44rem}
th,td{text-align:left; vertical-align:top; padding:.5rem .75rem .5rem 0;
  border-bottom:1px solid var(--rule-soft)}
th{font-family:var(--mono); font-size:11px; text-transform:uppercase;
  letter-spacing:.06em; color:var(--faint); font-weight:500; white-space:nowrap}
tbody tr:nth-child(odd){background:var(--cell)}
code,.m{font-family:var(--mono); font-size:12.5px}
td.m{white-space:nowrap; color:var(--signal)}
td.dim{color:var(--dim)}
ul.gaps{margin:.15rem 0 0; padding-left:1.1rem; color:var(--dim)}
.cmd{padding:1rem 0; border-bottom:1px solid var(--rule-soft)}
.cmd h3{font-family:var(--mono); font-size:13px; margin:0 0 .2rem; color:var(--signal); font-weight:500}
.cmd p{margin:0; color:var(--dim); font-size:13.5px}
.cmd table{margin-top:.6rem; min-width:0}
h3.sub{font-size:.95rem; margin:1.5rem 0 .5rem; font-weight:600}
footer{padding:2.5rem 0 4rem; color:var(--faint); font-size:13px}
footer p{max-width:52rem}
`

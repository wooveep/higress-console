function t(r){return(r||"").split(`
`).map(n=>n.trim()).filter(Boolean)}function i(r){return(r||[]).filter(n=>n!=null&&String(n).trim()).map(n=>String(n)).join(`
`)}function s(r,n){try{return JSON.parse(r)}catch{return n}}function e(r){return JSON.stringify(r??{},null,2)}export{s as a,t as b,i as j,e as s};

import{P as M}from"./PageSection-CvbT1lJk.js";import{L as O}from"./ListToolbar-COPCdeeQ.js";import{D as F}from"./DrawerFooter-DBGCpE2P.js";import{_ as z}from"./DeleteConfirmModal.vue_vue_type_script_setup_true_lang-CDV4Fn-w.js";import{d as L,o as d,e as C,f as c,t as P,v as b,B as h,w as s,i as N,y as U,r as p,C as B,_ as A,D as G,g as a,H as Y,J as H,E as q,j as Q,s as I}from"./index-DB8V18Gt.js";import{g as W,u as X,a as Z,d as ee}from"./llm-provider-BKQEr0bX.js";import{s as te,j as ae,a as oe,b as ne}from"./portal-BklEkoUq.js";import{_ as se,a as le}from"./index-xmjXpJFM.js";import{F as ie,_ as re}from"./index-BZkCDRfh.js";import{I as ue,a as pe}from"./index-Biqi-7uS.js";import{_ as me}from"./index-D52t1FaC.js";import"./hasIn-CtLKyq11.js";import"./useFlexGapSupport-Docsl1Vl.js";const ve={class:"secret-mask-text"},_e=L({__name:"SecretMaskText",props:{value:{}},setup(w){const l=w,r=p(!1),y=B(()=>l.value||"-"),m=B(()=>!l.value||r.value?y.value:l.value.length<=6?`${l.value.slice(0,1)}******`:`${l.value.slice(0,3)}******${l.value.slice(-3)}`);return(f,i)=>{const v=h;return d(),C("span",ve,[c("span",null,P(m.value),1),w.value?(d(),b(v,{key:0,type:"link",size:"small",class:"secret-mask-text__toggle",onClick:i[0]||(i[0]=n=>r.value=!r.value)},{default:s(()=>[N(P(r.value?"隐藏":"显示"),1)]),_:1})):U("",!0)])}}}),de=A(_e,[["__scopeId","data-v-b61bdab0"]]),fe={class:"provider-page__tokens"},ce={key:0},ye=`常用 rawConfigs 字段：
- providerDomain: 统一覆写上游域名
- providerBasePath: 统一追加上游基础路径，必须以 / 开头
- promoteThinkingOnEmpty: 在 content 为空时提升 reasoning_content
- hiclawMode: 开启后联动思维补齐能力
- bedrockPromptCachePointPositions: Bedrock Prompt Cache 注入位置
- promptCacheRetention: Bedrock 默认 prompt_cache_retention，支持 in_memory / 24h`,ge=`Vertex OAuth
{
  "vertexRegion": "asia-east1",
  "vertexProjectId": "demo-project",
  "vertexAuthKey": "{\\"type\\":\\"service_account\\",\\"client_email\\":\\"demo@example.com\\",\\"private_key_id\\":\\"key-id\\",\\"private_key\\":\\"-----BEGIN PRIVATE KEY-----\\\\n...\\\\n-----END PRIVATE KEY-----\\\\n\\",\\"token_uri\\":\\"https://oauth2.googleapis.com/token\\"}"
}

Vertex Express Mode(API Key)
{
  "vertexRegion": "asia-east1",
  "providerBasePath": "/v1beta1"
}

Claude / Gemini 自定义域名
{
  "providerDomain": "llm-proxy.example.com",
  "providerBasePath": "/anthropic"
}

Bedrock Prompt Cache
{
  "awsRegion": "us-west-2",
  "awsAccessKey": "AKIA...",
  "awsSecretKey": "secret",
  "promptCacheRetention": "in_memory",
  "bedrockPromptCachePointPositions": {
    "systemPrompt": true,
    "lastUserMessage": true
  }
}`,ke=L({__name:"ProviderPage",setup(w){const l=p(!1),r=p(""),y=p([]),m=p(!1),f=p(!1),i=p(null),v=p(null),n=Q({name:"",type:"",protocol:"openai/v1",proxyName:"",tokensText:"",rawConfigsJson:"{}"}),D=B(()=>y.value.filter(t=>{const e=r.value.trim().toLowerCase();return e?[t.name,t.type,t.protocol,t.proxyName].some(u=>String(u||"").toLowerCase().includes(e)):!0}));async function g(){l.value=!0;try{y.value=await W().catch(()=>[])}finally{l.value=!1}}function T(t){i.value=t||null,Object.assign(n,{name:(t==null?void 0:t.name)||"",type:(t==null?void 0:t.type)||"",protocol:(t==null?void 0:t.protocol)||"openai/v1",proxyName:(t==null?void 0:t.proxyName)||"",tokensText:ae(t==null?void 0:t.tokens),rawConfigsJson:te((t==null?void 0:t.rawConfigs)||{})}),m.value=!0}async function S(){var e;const t={...(e=i.value)!=null&&e.version?{version:i.value.version}:{},name:n.name,type:n.type,protocol:n.protocol,proxyName:n.proxyName||void 0,tokens:ne(n.tokensText),rawConfigs:oe(n.rawConfigsJson,{})};i.value?await X(t):await Z(t),m.value=!1,await g(),I("保存成功")}async function J(){v.value&&(await ee(v.value.name),f.value=!1,await g(),I("删除成功"))}return G(g),(t,e)=>{const u=le,$=h,R=se,k=ue,_=re,E=pe,V=me,K=ie,j=q;return d(),b(M,{title:"AI 服务提供者管理"},{default:s(()=>[a(O,{search:r.value,"onUpdate:search":e[0]||(e[0]=o=>r.value=o),"search-placeholder":"搜索名称、类型、协议","create-text":"新增 Provider",onRefresh:g,onCreate:e[1]||(e[1]=o=>T())},null,8,["search"]),a(R,{"data-source":D.value,loading:l.value,"row-key":"name",scroll:{x:980}},{default:s(()=>[a(u,{key:"type","data-index":"type",title:"类型"}),a(u,{key:"name","data-index":"name",title:"名称"}),a(u,{key:"protocol","data-index":"protocol",title:"协议"}),a(u,{key:"proxyName","data-index":"proxyName",title:"代理服务"}),a(u,{key:"tokens",title:"Tokens",width:"220"},{default:s(({record:o})=>[c("div",fe,[(d(!0),C(Y,null,H(o.tokens||[],x=>(d(),b(de,{key:x,value:x},null,8,["value"]))),128)),(o.tokens||[]).length?U("",!0):(d(),C("span",ce,"-"))])]),_:1}),a(u,{key:"actions",title:"操作",width:"180"},{default:s(({record:o})=>[a($,{type:"link",size:"small",onClick:x=>T(o)},{default:s(()=>[...e[11]||(e[11]=[N("编辑",-1)])]),_:1},8,["onClick"]),a($,{type:"link",size:"small",danger:"",onClick:x=>{v.value=o,f.value=!0}},{default:s(()=>[...e[12]||(e[12]=[N("删除",-1)])]),_:1},8,["onClick"])]),_:1})]),_:1},8,["data-source","loading"]),a(j,{open:m.value,"onUpdate:open":e[9]||(e[9]=o=>m.value=o),width:"720",title:i.value?"编辑 Provider":"新增 Provider"},{default:s(()=>[a(K,{layout:"vertical"},{default:s(()=>[a(_,{label:"名称"},{default:s(()=>[a(k,{value:n.name,"onUpdate:value":e[2]||(e[2]=o=>n.name=o),disabled:!!i.value},null,8,["value","disabled"])]),_:1}),a(_,{label:"类型"},{default:s(()=>[a(k,{value:n.type,"onUpdate:value":e[3]||(e[3]=o=>n.type=o)},null,8,["value"])]),_:1}),a(_,{label:"协议"},{default:s(()=>[a(k,{value:n.protocol,"onUpdate:value":e[4]||(e[4]=o=>n.protocol=o)},null,8,["value"])]),_:1}),a(_,{label:"代理服务"},{default:s(()=>[a(k,{value:n.proxyName,"onUpdate:value":e[5]||(e[5]=o=>n.proxyName=o)},null,8,["value"])]),_:1}),a(_,{label:"Tokens（一行一个）"},{default:s(()=>[a(E,{value:n.tokensText,"onUpdate:value":e[6]||(e[6]=o=>n.tokensText=o),rows:6},null,8,["value"])]),_:1}),a(_,{label:"rawConfigs(JSON)"},{default:s(()=>[a(E,{value:n.rawConfigsJson,"onUpdate:value":e[7]||(e[7]=o=>n.rawConfigsJson=o),rows:10,spellcheck:"false"},null,8,["value"]),a(V,{type:"info","show-icon":"",style:{"margin-top":"12px"},message:"rawConfigs 补充说明",description:ye}),c("div",{class:"provider-page__examples"},[e[13]||(e[13]=c("div",{class:"provider-page__examples-title"},"示例",-1)),c("pre",null,P(ge))])]),_:1})]),_:1}),a(F,{onCancel:e[8]||(e[8]=o=>m.value=!1),onConfirm:S})]),_:1},8,["open","title"]),a(z,{open:f.value,"onUpdate:open":e[10]||(e[10]=o=>f.value=o),content:v.value?`确认删除 ${v.value.name} 吗？`:"",onConfirm:J},null,8,["open","content"])]),_:1})}}}),Ue=A(ke,[["__scopeId","data-v-d780e8c9"]]);export{Ue as default};

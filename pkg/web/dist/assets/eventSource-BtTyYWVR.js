import{s as b,f as k}from"./index-Q9q6J7Kf.js";import{B as h,o as f,c as y,p as S,x as d,z as w,d as v,y as $,a as P,t as T,r as g}from"./index-Dwm6kBzc.js";var B=`
    .p-tag {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        background: dt('tag.primary.background');
        color: dt('tag.primary.color');
        font-size: dt('tag.font.size');
        font-weight: dt('tag.font.weight');
        padding: dt('tag.padding');
        border-radius: dt('tag.border.radius');
        gap: dt('tag.gap');
    }

    .p-tag-icon {
        font-size: dt('tag.icon.size');
        width: dt('tag.icon.size');
        height: dt('tag.icon.size');
    }

    .p-tag-rounded {
        border-radius: dt('tag.rounded.border.radius');
    }

    .p-tag-success {
        background: dt('tag.success.background');
        color: dt('tag.success.color');
    }

    .p-tag-info {
        background: dt('tag.info.background');
        color: dt('tag.info.color');
    }

    .p-tag-warn {
        background: dt('tag.warn.background');
        color: dt('tag.warn.color');
    }

    .p-tag-danger {
        background: dt('tag.danger.background');
        color: dt('tag.danger.color');
    }

    .p-tag-secondary {
        background: dt('tag.secondary.background');
        color: dt('tag.secondary.color');
    }

    .p-tag-contrast {
        background: dt('tag.contrast.background');
        color: dt('tag.contrast.color');
    }
`,z={root:function(n){var t=n.props;return["p-tag p-component",{"p-tag-info":t.severity==="info","p-tag-success":t.severity==="success","p-tag-warn":t.severity==="warn","p-tag-danger":t.severity==="danger","p-tag-secondary":t.severity==="secondary","p-tag-contrast":t.severity==="contrast","p-tag-rounded":t.rounded}]},icon:"p-tag-icon",label:"p-tag-label"},E=h.extend({name:"tag",style:B,classes:z}),C={name:"BaseTag",extends:b,props:{value:null,severity:null,rounded:Boolean,icon:String},style:E,provide:function(){return{$pcTag:this,$parentInstance:this}}};function l(e){"@babel/helpers - typeof";return l=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(n){return typeof n}:function(n){return n&&typeof Symbol=="function"&&n.constructor===Symbol&&n!==Symbol.prototype?"symbol":typeof n},l(e)}function j(e,n,t){return(n=D(n))in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function D(e){var n=N(e,"string");return l(n)=="symbol"?n:n+""}function N(e,n){if(l(e)!="object"||!e)return e;var t=e[Symbol.toPrimitive];if(t!==void 0){var r=t.call(e,n);if(l(r)!="object")return r;throw new TypeError("@@toPrimitive must return a primitive value.")}return(n==="string"?String:Number)(e)}var O={name:"Tag",extends:C,inheritAttrs:!1,computed:{dataP:function(){return k(j({rounded:this.rounded},this.severity,this.severity))}}},A=["data-p"];function M(e,n,t,r,s,a){return f(),y("span",d({class:e.cx("root"),"data-p":a.dataP},e.ptmi("root")),[e.$slots.icon?(f(),S(w(e.$slots.icon),d({key:0,class:e.cx("icon")},e.ptm("icon")),null,16,["class"])):e.icon?(f(),y("span",d({key:1,class:[e.cx("icon"),e.icon]},e.ptm("icon")),null,16)):v("",!0),e.value!=null||e.$slots.default?$(e.$slots,"default",{key:2},function(){return[P("span",d({class:e.cx("label")},e.ptm("label")),T(e.value),17)]}):v("",!0)],16,A)}O.render=M;function J(e,n={}){const t=g(null),r=g(null),s=g(!1),a=g(null);let i=0,o=null;const p=()=>{o&&(clearTimeout(o),o=null);try{a.value=new EventSource(e,{withCredentials:!0,...n}),a.value.onopen=()=>{s.value=!0,r.value=null,i=0},a.value.onmessage=c=>{try{const u=JSON.parse(c.data);t.value=u}catch{t.value=c.data}},a.value.onerror=c=>{if(s.value=!1,r.value=c,a.value.readyState===EventSource.CLOSED)return;const u=Math.min(1e3*Math.pow(2,i),3e4);i++,o=setTimeout(()=>{p()},u)}}catch(c){r.value=c,s.value=!1}},m=()=>{o&&(clearTimeout(o),o=null),a.value&&(a.value.close(),a.value=null,s.value=!1,i=0)};return p(),{data:t,error:r,isConnected:s,disconnect:m,reconnect:p}}export{O as s,J as u};

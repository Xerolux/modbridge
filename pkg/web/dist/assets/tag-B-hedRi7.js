import{Bt as o,Dt as l,Et as u,Ot as i,Wt as s,Xt as g,kt as c,qt as p,t as y,yn as f}from"./style-Cr3jq0ZU.js";import{i as b,n as m}from"./baseicon-FLrGHloj.js";var v=`
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
`,k=y.extend({name:"tag",style:v,classes:{root:function(t){var r=t.props;return["p-tag p-component",{"p-tag-info":r.severity==="info","p-tag-success":r.severity==="success","p-tag-warn":r.severity==="warn","p-tag-danger":r.severity==="danger","p-tag-secondary":r.severity==="secondary","p-tag-contrast":r.severity==="contrast","p-tag-rounded":r.rounded}]},icon:"p-tag-icon",label:"p-tag-label"}}),h={name:"BaseTag",extends:m,props:{value:null,severity:null,rounded:Boolean,icon:String},style:k,provide:function(){return{$pcTag:this,$parentInstance:this}}};function e(n){"@babel/helpers - typeof";return e=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(t){return typeof t}:function(t){return t&&typeof Symbol=="function"&&t.constructor===Symbol&&t!==Symbol.prototype?"symbol":typeof t},e(n)}function S(n,t,r){return(t=w(t))in n?Object.defineProperty(n,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):n[t]=r,n}function w(n){var t=$(n,"string");return e(t)=="symbol"?t:t+""}function $(n,t){if(e(n)!="object"||!n)return n;var r=n[Symbol.toPrimitive];if(r!==void 0){var a=r.call(n,t);if(e(a)!="object")return a;throw new TypeError("@@toPrimitive must return a primitive value.")}return(t==="string"?String:Number)(n)}var P={name:"Tag",extends:h,inheritAttrs:!1,computed:{dataP:function(){return b(S({rounded:this.rounded},this.severity,this.severity))}}},B=["data-p"];function z(n,t,r,a,T,d){return s(),c("span",o({class:n.cx("root"),"data-p":d.dataP},n.ptmi("root")),[n.$slots.icon?(s(),l(g(n.$slots.icon),o({key:0,class:n.cx("icon")},n.ptm("icon")),null,16,["class"])):n.icon?(s(),c("span",o({key:1,class:[n.cx("icon"),n.icon]},n.ptm("icon")),null,16)):i("",!0),n.value!=null||n.$slots.default?p(n.$slots,"default",{key:2},function(){return[u("span",o({class:n.cx("label")},n.ptm("label")),f(n.value),17)]}):i("",!0)],16,B)}P.render=z;export{P as t};

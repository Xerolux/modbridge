import{Bt as i,Dt as k,Et as d,Ht as B,Jt as V,Mt as D,Nt as h,Ot as b,Wt as a,Xt as C,Yt as E,bn as N,bt as S,en as x,gn as T,hn as p,kt as l,mt as L,qt as w,t as A,tn as K,un as f,yn as M}from"./style-Cr3jq0ZU.js";import{i as U,n as W}from"./baseicon-FLrGHloj.js";import{a as R,n as q,t as H}from"./times-BWuEkU-k.js";import{a as J,r as X}from"./index-Ca-AG6VN.js";import"./baseeditableholder-jlPCrbKC.js";import{t as Y}from"./plugin-vue_export-helper-BAVuyXO6.js";import"./baseinput-Ha0RXyiH.js";import{t as j}from"./inputtext-BPu-iRjy.js";import{t as F}from"./card-DQs0Bwk0.js";var G=`
    .p-message {
        display: grid;
        grid-template-rows: 1fr;
        border-radius: dt('message.border.radius');
        outline-width: dt('message.border.width');
        outline-style: solid;
    }

    .p-message-content-wrapper {
        min-height: 0;
    }

    .p-message-content {
        display: flex;
        align-items: center;
        padding: dt('message.content.padding');
        gap: dt('message.content.gap');
    }

    .p-message-icon {
        flex-shrink: 0;
    }

    .p-message-close-button {
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        margin-inline-start: auto;
        overflow: hidden;
        position: relative;
        width: dt('message.close.button.width');
        height: dt('message.close.button.height');
        border-radius: dt('message.close.button.border.radius');
        background: transparent;
        transition:
            background dt('message.transition.duration'),
            color dt('message.transition.duration'),
            outline-color dt('message.transition.duration'),
            box-shadow dt('message.transition.duration'),
            opacity 0.3s;
        outline-color: transparent;
        color: inherit;
        padding: 0;
        border: none;
        cursor: pointer;
        user-select: none;
    }

    .p-message-close-icon {
        font-size: dt('message.close.icon.size');
        width: dt('message.close.icon.size');
        height: dt('message.close.icon.size');
    }

    .p-message-close-button:focus-visible {
        outline-width: dt('message.close.button.focus.ring.width');
        outline-style: dt('message.close.button.focus.ring.style');
        outline-offset: dt('message.close.button.focus.ring.offset');
    }

    .p-message-info {
        background: dt('message.info.background');
        outline-color: dt('message.info.border.color');
        color: dt('message.info.color');
        box-shadow: dt('message.info.shadow');
    }

    .p-message-info .p-message-close-button:focus-visible {
        outline-color: dt('message.info.close.button.focus.ring.color');
        box-shadow: dt('message.info.close.button.focus.ring.shadow');
    }

    .p-message-info .p-message-close-button:hover {
        background: dt('message.info.close.button.hover.background');
    }

    .p-message-info.p-message-outlined {
        color: dt('message.info.outlined.color');
        outline-color: dt('message.info.outlined.border.color');
    }

    .p-message-info.p-message-simple {
        color: dt('message.info.simple.color');
    }

    .p-message-success {
        background: dt('message.success.background');
        outline-color: dt('message.success.border.color');
        color: dt('message.success.color');
        box-shadow: dt('message.success.shadow');
    }

    .p-message-success .p-message-close-button:focus-visible {
        outline-color: dt('message.success.close.button.focus.ring.color');
        box-shadow: dt('message.success.close.button.focus.ring.shadow');
    }

    .p-message-success .p-message-close-button:hover {
        background: dt('message.success.close.button.hover.background');
    }

    .p-message-success.p-message-outlined {
        color: dt('message.success.outlined.color');
        outline-color: dt('message.success.outlined.border.color');
    }

    .p-message-success.p-message-simple {
        color: dt('message.success.simple.color');
    }

    .p-message-warn {
        background: dt('message.warn.background');
        outline-color: dt('message.warn.border.color');
        color: dt('message.warn.color');
        box-shadow: dt('message.warn.shadow');
    }

    .p-message-warn .p-message-close-button:focus-visible {
        outline-color: dt('message.warn.close.button.focus.ring.color');
        box-shadow: dt('message.warn.close.button.focus.ring.shadow');
    }

    .p-message-warn .p-message-close-button:hover {
        background: dt('message.warn.close.button.hover.background');
    }

    .p-message-warn.p-message-outlined {
        color: dt('message.warn.outlined.color');
        outline-color: dt('message.warn.outlined.border.color');
    }

    .p-message-warn.p-message-simple {
        color: dt('message.warn.simple.color');
    }

    .p-message-error {
        background: dt('message.error.background');
        outline-color: dt('message.error.border.color');
        color: dt('message.error.color');
        box-shadow: dt('message.error.shadow');
    }

    .p-message-error .p-message-close-button:focus-visible {
        outline-color: dt('message.error.close.button.focus.ring.color');
        box-shadow: dt('message.error.close.button.focus.ring.shadow');
    }

    .p-message-error .p-message-close-button:hover {
        background: dt('message.error.close.button.hover.background');
    }

    .p-message-error.p-message-outlined {
        color: dt('message.error.outlined.color');
        outline-color: dt('message.error.outlined.border.color');
    }

    .p-message-error.p-message-simple {
        color: dt('message.error.simple.color');
    }

    .p-message-secondary {
        background: dt('message.secondary.background');
        outline-color: dt('message.secondary.border.color');
        color: dt('message.secondary.color');
        box-shadow: dt('message.secondary.shadow');
    }

    .p-message-secondary .p-message-close-button:focus-visible {
        outline-color: dt('message.secondary.close.button.focus.ring.color');
        box-shadow: dt('message.secondary.close.button.focus.ring.shadow');
    }

    .p-message-secondary .p-message-close-button:hover {
        background: dt('message.secondary.close.button.hover.background');
    }

    .p-message-secondary.p-message-outlined {
        color: dt('message.secondary.outlined.color');
        outline-color: dt('message.secondary.outlined.border.color');
    }

    .p-message-secondary.p-message-simple {
        color: dt('message.secondary.simple.color');
    }

    .p-message-contrast {
        background: dt('message.contrast.background');
        outline-color: dt('message.contrast.border.color');
        color: dt('message.contrast.color');
        box-shadow: dt('message.contrast.shadow');
    }

    .p-message-contrast .p-message-close-button:focus-visible {
        outline-color: dt('message.contrast.close.button.focus.ring.color');
        box-shadow: dt('message.contrast.close.button.focus.ring.shadow');
    }

    .p-message-contrast .p-message-close-button:hover {
        background: dt('message.contrast.close.button.hover.background');
    }

    .p-message-contrast.p-message-outlined {
        color: dt('message.contrast.outlined.color');
        outline-color: dt('message.contrast.outlined.border.color');
    }

    .p-message-contrast.p-message-simple {
        color: dt('message.contrast.simple.color');
    }

    .p-message-text {
        font-size: dt('message.text.font.size');
        font-weight: dt('message.text.font.weight');
    }

    .p-message-icon {
        font-size: dt('message.icon.size');
        width: dt('message.icon.size');
        height: dt('message.icon.size');
    }

    .p-message-sm .p-message-content {
        padding: dt('message.content.sm.padding');
    }

    .p-message-sm .p-message-text {
        font-size: dt('message.text.sm.font.size');
    }

    .p-message-sm .p-message-icon {
        font-size: dt('message.icon.sm.size');
        width: dt('message.icon.sm.size');
        height: dt('message.icon.sm.size');
    }

    .p-message-sm .p-message-close-icon {
        font-size: dt('message.close.icon.sm.size');
        width: dt('message.close.icon.sm.size');
        height: dt('message.close.icon.sm.size');
    }

    .p-message-lg .p-message-content {
        padding: dt('message.content.lg.padding');
    }

    .p-message-lg .p-message-text {
        font-size: dt('message.text.lg.font.size');
    }

    .p-message-lg .p-message-icon {
        font-size: dt('message.icon.lg.size');
        width: dt('message.icon.lg.size');
        height: dt('message.icon.lg.size');
    }

    .p-message-lg .p-message-close-icon {
        font-size: dt('message.close.icon.lg.size');
        width: dt('message.close.icon.lg.size');
        height: dt('message.close.icon.lg.size');
    }

    .p-message-outlined {
        background: transparent;
        outline-width: dt('message.outlined.border.width');
    }

    .p-message-simple {
        background: transparent;
        outline-color: transparent;
        box-shadow: none;
    }

    .p-message-simple .p-message-content {
        padding: dt('message.simple.content.padding');
    }

    .p-message-outlined .p-message-close-button:hover,
    .p-message-simple .p-message-close-button:hover {
        background: transparent;
    }

    .p-message-enter-active {
        animation: p-animate-message-enter 0.3s ease-out forwards;
        overflow: hidden;
    }

    .p-message-leave-active {
        animation: p-animate-message-leave 0.15s ease-in forwards;
        overflow: hidden;
    }

    @keyframes p-animate-message-enter {
        from {
            opacity: 0;
            grid-template-rows: 0fr;
        }
        to {
            opacity: 1;
            grid-template-rows: 1fr;
        }
    }

    @keyframes p-animate-message-leave {
        from {
            opacity: 1;
            grid-template-rows: 1fr;
        }
        to {
            opacity: 0;
            margin: 0;
            grid-template-rows: 0fr;
        }
    }
`,Q=A.extend({name:"message",style:G,classes:{root:function(s){var n=s.props;return["p-message p-component p-message-"+n.severity,{"p-message-outlined":n.variant==="outlined","p-message-simple":n.variant==="simple","p-message-sm":n.size==="small","p-message-lg":n.size==="large"}]},contentWrapper:"p-message-content-wrapper",content:"p-message-content",icon:"p-message-icon",text:"p-message-text",closeButton:"p-message-close-button",closeIcon:"p-message-close-icon"}}),Z={name:"BaseMessage",extends:W,props:{severity:{type:String,default:"info"},closable:{type:Boolean,default:!1},life:{type:Number,default:null},icon:{type:String,default:void 0},closeIcon:{type:String,default:void 0},closeButtonProps:{type:null,default:null},size:{type:String,default:null},variant:{type:String,default:null}},style:Q,provide:function(){return{$pcMessage:this,$parentInstance:this}}};function v(e){"@babel/helpers - typeof";return v=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(s){return typeof s}:function(s){return s&&typeof Symbol=="function"&&s.constructor===Symbol&&s!==Symbol.prototype?"symbol":typeof s},v(e)}function O(e,s,n){return(s=ee(s))in e?Object.defineProperty(e,s,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[s]=n,e}function ee(e){var s=se(e,"string");return v(s)=="symbol"?s:s+""}function se(e,s){if(v(e)!="object"||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var o=n.call(e,s);if(v(o)!="object")return o;throw new TypeError("@@toPrimitive must return a primitive value.")}return(s==="string"?String:Number)(e)}var I={name:"Message",extends:Z,inheritAttrs:!1,emits:["close","life-end"],timeout:null,data:function(){return{visible:!0}},mounted:function(){var s=this;this.life&&setTimeout(function(){s.visible=!1,s.$emit("life-end")},this.life)},methods:{close:function(s){this.visible=!1,this.$emit("close",s)}},computed:{closeAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.close:void 0},dataP:function(){return U(O(O({outlined:this.variant==="outlined",simple:this.variant==="simple"},this.severity,this.severity),this.size,this.size))}},directives:{ripple:R},components:{TimesIcon:H}};function y(e){"@babel/helpers - typeof";return y=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(s){return typeof s}:function(s){return s&&typeof Symbol=="function"&&s.constructor===Symbol&&s!==Symbol.prototype?"symbol":typeof s},y(e)}function $(e,s){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);s&&(o=o.filter(function(u){return Object.getOwnPropertyDescriptor(e,u).enumerable})),n.push.apply(n,o)}return n}function _(e){for(var s=1;s<arguments.length;s++){var n=arguments[s]!=null?arguments[s]:{};s%2?$(Object(n),!0).forEach(function(o){ne(e,o,n[o])}):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):$(Object(n)).forEach(function(o){Object.defineProperty(e,o,Object.getOwnPropertyDescriptor(n,o))})}return e}function ne(e,s,n){return(s=oe(s))in e?Object.defineProperty(e,s,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[s]=n,e}function oe(e){var s=te(e,"string");return y(s)=="symbol"?s:s+""}function te(e,s){if(y(e)!="object"||!e)return e;var n=e[Symbol.toPrimitive];if(n!==void 0){var o=n.call(e,s);if(y(o)!="object")return o;throw new TypeError("@@toPrimitive must return a primitive value.")}return(s==="string"?String:Number)(e)}var ae=["data-p"],re=["data-p"],ie=["data-p"],le=["aria-label","data-p"],ce=["data-p"];function de(e,s,n,o,u,r){var m=V("TimesIcon"),c=E("ripple");return a(),k(L,i({name:"p-message",appear:""},e.ptmi("transition")),{default:x(function(){return[u.visible?(a(),l("div",i({key:0,class:e.cx("root"),role:"alert","aria-live":"assertive","aria-atomic":"true","data-p":r.dataP},e.ptm("root")),[d("div",i({class:e.cx("contentWrapper")},e.ptm("contentWrapper")),[e.$slots.container?w(e.$slots,"container",{key:0,closeCallback:r.close}):(a(),l("div",i({key:1,class:e.cx("content"),"data-p":r.dataP},e.ptm("content")),[w(e.$slots,"icon",{class:T(e.cx("icon"))},function(){return[(a(),k(C(e.icon?"span":null),i({class:[e.cx("icon"),e.icon],"data-p":r.dataP},e.ptm("icon")),null,16,["class","data-p"]))]}),e.$slots.default?(a(),l("div",i({key:0,class:e.cx("text"),"data-p":r.dataP},e.ptm("text")),[w(e.$slots,"default")],16,ie)):b("",!0),e.closable?K((a(),l("button",i({key:1,class:e.cx("closeButton"),"aria-label":r.closeAriaLabel,type:"button",onClick:s[0]||(s[0]=function(g){return r.close(g)}),"data-p":r.dataP},_(_({},e.closeButtonProps),e.ptm("closeButton"))),[w(e.$slots,"closeicon",{},function(){return[e.closeIcon?(a(),l("i",i({key:0,class:[e.cx("closeIcon"),e.closeIcon],"data-p":r.dataP},e.ptm("closeIcon")),null,16,ce)):(a(),k(m,i({key:1,class:[e.cx("closeIcon"),e.closeIcon],"data-p":r.dataP},e.ptm("closeIcon")),null,16,["class","data-p"]))]})],16,le)),[[c]]):b("",!0)],16,re))],16)],16,ae)):b("",!0)]}),_:3},16)}I.render=de;var ue={class:"flex items-center justify-center min-h-[80vh] px-4 py-8"},me={class:"text-sm font-normal text-surface-400"},ge={key:0},pe={key:1},fe={class:"flex flex-col gap-5 mt-4"},be={key:0,class:"flex flex-col gap-2"},ve={class:"flex flex-col gap-2"},ye={__name:"Login",setup(e){const s=f(""),n=f(""),o=f(""),u=X(),r=J(),m=f(!1),c=f(!1);B(async()=>{try{c.value=(await N.get("/api/status",{skipAuth:!0})).data.multi_user===!0}catch{c.value=!1}});const g=async()=>{m.value=!0,o.value="";const z={password:n.value};c.value&&(z.username=s.value);const t=await u.login(z);m.value=!1,t.success?r.push("/"):o.value=t.message||"Invalid credentials"};return(z,t)=>(a(),l("div",ue,[h(p(F),{class:"w-full max-w-md glass-card border border-white/10 shadow-2xl overflow-hidden relative"},{title:x(()=>[t[2]||(t[2]=d("div",{class:"text-2xl font-semibold tracking-tight text-white mb-2"},"Welcome Back",-1)),d("div",me,[c.value?(a(),l("span",ge,"Enter your credentials to continue")):(a(),l("span",pe,"Enter your password to continue"))])]),content:x(()=>[d("div",fe,[c.value?(a(),l("div",be,[t[3]||(t[3]=d("label",{for:"username",class:"text-sm font-medium text-surface-200"},"Username",-1)),h(p(j),{id:"username",modelValue:s.value,"onUpdate:modelValue":t[0]||(t[0]=P=>s.value=P),onKeyup:S(g,["enter"]),class:"p-3 w-full bg-surface-800/50 border-surface-700/50 text-white focus:border-primary-500 transition-colors rounded-xl",placeholder:"Username"},null,8,["modelValue"])])):b("",!0),d("div",ve,[t[4]||(t[4]=d("label",{for:"password",class:"text-sm font-medium text-surface-200"},"Password",-1)),h(p(j),{id:"password",modelValue:n.value,"onUpdate:modelValue":t[1]||(t[1]=P=>n.value=P),type:"password",onKeyup:S(g,["enter"]),class:"p-3 w-full bg-surface-800/50 border-surface-700/50 text-white focus:border-primary-500 transition-colors rounded-xl",placeholder:"••••••••"},null,8,["modelValue"])]),o.value?(a(),k(p(I),{key:1,severity:"error",class:"text-sm rounded-xl"},{default:x(()=>[D(M(o.value),1)]),_:1})):b("",!0),h(p(q),{label:"Login",onClick:g,loading:m.value,class:"btn-neon w-full p-3 font-semibold mt-2 rounded-xl"},null,8,["loading"])])]),_:1})]))}},$e=Y(ye,[["__scopeId","data-v-1bc41d6e"]]);export{$e as default};

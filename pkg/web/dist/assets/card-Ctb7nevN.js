import{Et as n,Kt as a,Ot as d,Ut as r,kt as o,t as s,zt as e}from"./style-njBsFZ_t.js";import{n as i}from"./baseicon-Cp-AYqdT.js";var l=`
    .p-card {
        background: dt('card.background');
        color: dt('card.color');
        box-shadow: dt('card.shadow');
        border-radius: dt('card.border.radius');
        display: flex;
        flex-direction: column;
    }

    .p-card-caption {
        display: flex;
        flex-direction: column;
        gap: dt('card.caption.gap');
    }

    .p-card-body {
        padding: dt('card.body.padding');
        display: flex;
        flex-direction: column;
        gap: dt('card.body.gap');
    }

    .p-card-title {
        font-size: dt('card.title.font.size');
        font-weight: dt('card.title.font.weight');
    }

    .p-card-subtitle {
        color: dt('card.subtitle.color');
    }
`,p=s.extend({name:"card",style:l,classes:{root:"p-card p-component",header:"p-card-header",body:"p-card-body",caption:"p-card-caption",title:"p-card-title",subtitle:"p-card-subtitle",content:"p-card-content",footer:"p-card-footer"}}),c={name:"BaseCard",extends:i,style:p,provide:function(){return{$pcCard:this,$parentInstance:this}}},u={name:"Card",extends:c,inheritAttrs:!1};function m(t,b,f,y,$,h){return r(),o("div",e({class:t.cx("root")},t.ptmi("root")),[t.$slots.header?(r(),o("div",e({key:0,class:t.cx("header")},t.ptm("header")),[a(t.$slots,"header")],16)):d("",!0),n("div",e({class:t.cx("body")},t.ptm("body")),[t.$slots.title||t.$slots.subtitle?(r(),o("div",e({key:0,class:t.cx("caption")},t.ptm("caption")),[t.$slots.title?(r(),o("div",e({key:0,class:t.cx("title")},t.ptm("title")),[a(t.$slots,"title")],16)):d("",!0),t.$slots.subtitle?(r(),o("div",e({key:1,class:t.cx("subtitle")},t.ptm("subtitle")),[a(t.$slots,"subtitle")],16)):d("",!0)],16)):d("",!0),n("div",e({class:t.cx("content")},t.ptm("content")),[a(t.$slots,"content")],16),t.$slots.footer?(r(),o("div",e({key:1,class:t.cx("footer")},t.ptm("footer")),[a(t.$slots,"footer")],16)):d("",!0)],16)],16)}u.render=m;export{u as t};

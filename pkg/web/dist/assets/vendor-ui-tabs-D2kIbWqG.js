import{C as I,Ct as h,Et as P,Gt as N,H as S,Hn as C,Ht as A,In as d,J as v,Nn as l,Ot as x,Rn as R,Un as g,Wt as w,Zt as V,_n as B,_t as c,c as W,fn as T,gn as f,ir as _,kn as i,ln as E,nt as p,ut as H,vn as m,yn as b,zn as k}from"./vendor-ui-core-CBrKke27.js";var D=`
    .p-tabs {
        display: flex;
        flex-direction: column;
    }

    .p-tablist {
        display: flex;
        position: relative;
        overflow: hidden;
        background: dt('tabs.tablist.background');
    }

    .p-tablist-viewport {
        overflow-x: auto;
        overflow-y: hidden;
        scroll-behavior: smooth;
        scrollbar-width: none;
        overscroll-behavior: contain auto;
    }

    .p-tablist-viewport::-webkit-scrollbar {
        display: none;
    }

    .p-tablist-tab-list {
        position: relative;
        display: flex;
        border-style: solid;
        border-color: dt('tabs.tablist.border.color');
        border-width: dt('tabs.tablist.border.width');
    }

    .p-tablist-content {
        flex-grow: 1;
    }

    .p-tablist-nav-button {
        all: unset;
        position: absolute !important;
        flex-shrink: 0;
        inset-block-start: 0;
        z-index: 2;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: dt('tabs.nav.button.background');
        color: dt('tabs.nav.button.color');
        width: dt('tabs.nav.button.width');
        transition:
            color dt('tabs.transition.duration'),
            outline-color dt('tabs.transition.duration'),
            box-shadow dt('tabs.transition.duration');
        box-shadow: dt('tabs.nav.button.shadow');
        outline-color: transparent;
        cursor: pointer;
    }

    .p-tablist-nav-button:focus-visible {
        z-index: 1;
        box-shadow: dt('tabs.nav.button.focus.ring.shadow');
        outline: dt('tabs.nav.button.focus.ring.width') dt('tabs.nav.button.focus.ring.style') dt('tabs.nav.button.focus.ring.color');
        outline-offset: dt('tabs.nav.button.focus.ring.offset');
    }

    .p-tablist-nav-button:hover {
        color: dt('tabs.nav.button.hover.color');
    }

    .p-tablist-prev-button {
        inset-inline-start: 0;
    }

    .p-tablist-next-button {
        inset-inline-end: 0;
    }

    .p-tablist-prev-button:dir(rtl),
    .p-tablist-next-button:dir(rtl) {
        transform: rotate(180deg);
    }

    .p-tab {
        flex-shrink: 0;
        cursor: pointer;
        user-select: none;
        position: relative;
        border-style: solid;
        white-space: nowrap;
        gap: dt('tabs.tab.gap');
        background: dt('tabs.tab.background');
        border-width: dt('tabs.tab.border.width');
        border-color: dt('tabs.tab.border.color');
        color: dt('tabs.tab.color');
        padding: dt('tabs.tab.padding');
        font-weight: dt('tabs.tab.font.weight');
        transition:
            background dt('tabs.transition.duration'),
            border-color dt('tabs.transition.duration'),
            color dt('tabs.transition.duration'),
            outline-color dt('tabs.transition.duration'),
            box-shadow dt('tabs.transition.duration');
        margin: dt('tabs.tab.margin');
        outline-color: transparent;
    }

    .p-tab:not(.p-disabled):focus-visible {
        z-index: 1;
        box-shadow: dt('tabs.tab.focus.ring.shadow');
        outline: dt('tabs.tab.focus.ring.width') dt('tabs.tab.focus.ring.style') dt('tabs.tab.focus.ring.color');
        outline-offset: dt('tabs.tab.focus.ring.offset');
    }

    .p-tab:not(.p-tab-active):not(.p-disabled):hover {
        background: dt('tabs.tab.hover.background');
        border-color: dt('tabs.tab.hover.border.color');
        color: dt('tabs.tab.hover.color');
    }

    .p-tab-active {
        background: dt('tabs.tab.active.background');
        border-color: dt('tabs.tab.active.border.color');
        color: dt('tabs.tab.active.color');
    }

    .p-tabpanels {
        background: dt('tabs.tabpanel.background');
        color: dt('tabs.tabpanel.color');
        padding: dt('tabs.tabpanel.padding');
        outline: 0 none;
    }

    .p-tabpanel:focus-visible {
        box-shadow: dt('tabs.tabpanel.focus.ring.shadow');
        outline: dt('tabs.tabpanel.focus.ring.width') dt('tabs.tabpanel.focus.ring.style') dt('tabs.tabpanel.focus.ring.color');
        outline-offset: dt('tabs.tabpanel.focus.ring.offset');
    }

    .p-tablist-active-bar {
        z-index: 1;
        display: block;
        position: absolute;
        inset-block-end: dt('tabs.active.bar.bottom');
        height: dt('tabs.active.bar.height');
        background: dt('tabs.active.bar.background');
        transition: 250ms cubic-bezier(0.35, 0, 0.25, 1);
    }
`,M=p.extend({name:"tabs",style:D,classes:{root:function(t){return["p-tabs p-component",{"p-tabs-scrollable":t.props.scrollable}]}}}),j={name:"Tabs",extends:{name:"BaseTabs",extends:v,props:{value:{type:[String,Number],default:void 0},lazy:{type:Boolean,default:!1},scrollable:{type:Boolean,default:!1},showNavigators:{type:Boolean,default:!0},tabindex:{type:Number,default:0},selectOnFocus:{type:Boolean,default:!1}},style:M,provide:function(){return{$pcTabs:this,$parentInstance:this}}},inheritAttrs:!1,emits:["update:value"],data:function(){return{d_value:this.value}},watch:{value:function(t){this.d_value=t}},methods:{updateValue:function(t){this.d_value!==t&&(this.d_value=t,this.$emit("update:value",t))},isVertical:function(){return this.orientation==="vertical"}}};function U(e,t,r,a,s,n){return l(),b("div",i({class:e.cx("root")},e.ptmi("root")),[d(e.$slots,"default")],16)}j.render=U;var F=p.extend({name:"tablist",classes:{root:"p-tablist",content:"p-tablist-content p-tablist-viewport",tabList:"p-tablist-tab-list",activeBar:"p-tablist-active-bar",prevButton:"p-tablist-prev-button p-tablist-nav-button",nextButton:"p-tablist-next-button p-tablist-nav-button"}}),G={name:"TabList",extends:{name:"BaseTabList",extends:v,props:{},style:F,provide:function(){return{$pcTabList:this,$parentInstance:this}}},inheritAttrs:!1,inject:["$pcTabs"],data:function(){return{isPrevButtonEnabled:!1,isNextButtonEnabled:!0}},resizeObserver:void 0,inkBarObserver:void 0,watch:{showNavigators:function(t){t?this.bindResizeObserver():this.unbindResizeObserver()},activeValue:{flush:"post",handler:function(){this.updateInkBar(),this.bindInkBarObserver()}}},mounted:function(){var t=this;setTimeout(function(){t.updateInkBar(),t.bindInkBarObserver()},150),this.showNavigators&&(this.updateButtonState(),this.bindResizeObserver())},updated:function(){this.showNavigators&&this.updateButtonState()},beforeUnmount:function(){this.unbindResizeObserver(),this.unbindInkBarObserver()},methods:{onScroll:function(t){this.showNavigators&&this.updateButtonState(),t.preventDefault()},onPrevButtonClick:function(){var t=this.$refs.content,r=this.getVisibleButtonWidths(),a=h(t)-r,s=Math.abs(t.scrollLeft)-a*.8,n=Math.max(s,0);t.scrollLeft=x(t)?-1*n:n},onNextButtonClick:function(){var t=this.$refs.content,r=this.getVisibleButtonWidths(),a=h(t)-r,s=Math.abs(t.scrollLeft)+a*.8,n=t.scrollWidth-a,o=Math.min(s,n);t.scrollLeft=x(t)?-1*o:o},bindResizeObserver:function(){var t=this;this.resizeObserver=new ResizeObserver(function(){return t.updateButtonState()}),this.resizeObserver.observe(this.$refs.list)},unbindResizeObserver:function(){var t;(t=this.resizeObserver)===null||t===void 0||t.unobserve(this.$refs.list),this.resizeObserver=void 0},bindInkBarObserver:function(){var t=this;this.unbindInkBarObserver();var r=this.$refs.content,a=w(r,'[data-pc-name="tab"][data-p-active="true"]');a&&(this.inkBarObserver=new ResizeObserver(function(){return t.updateInkBar()}),this.inkBarObserver.observe(a))},unbindInkBarObserver:function(){var t;(t=this.inkBarObserver)===null||t===void 0||t.disconnect(),this.inkBarObserver=void 0},updateInkBar:function(){var t=this.$refs,r=t.content,a=t.inkbar,s=t.tabs;if(a){var n=w(r,'[data-pc-name="tab"][data-p-active="true"]');this.$pcTabs.isVertical()?(a.style.height=H(n)+"px",a.style.top=c(n).top-c(s).top+"px"):(a.style.width=A(n)+"px",a.style.left=c(n).left-c(s).left+"px")}},updateButtonState:function(){var t=this.$refs,r=t.list,a=t.content,s=a.scrollTop,n=a.scrollWidth,o=a.scrollHeight,u=a.offsetWidth,O=a.offsetHeight,$=Math.abs(a.scrollLeft),y=[h(a),P(a)],L=y[0],z=y[1];this.$pcTabs.isVertical()?(this.isPrevButtonEnabled=s!==0,this.isNextButtonEnabled=r.offsetHeight>=O&&parseInt(s)!==o-z):(this.isPrevButtonEnabled=$!==0,this.isNextButtonEnabled=r.offsetWidth>=u&&parseInt($)!==n-L)},getVisibleButtonWidths:function(){var t=this.$refs,r=t.prevButton,a=t.nextButton,s=0;return this.showNavigators&&(s=(r?.offsetWidth||0)+(a?.offsetWidth||0)),s}},computed:{templates:function(){return this.$pcTabs.$slots},activeValue:function(){return this.$pcTabs.d_value},showNavigators:function(){return this.$pcTabs.showNavigators},prevButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.previous:void 0},nextButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.next:void 0},dataP:function(){return N({scrollable:this.$pcTabs.scrollable})}},components:{ChevronLeftIcon:W,ChevronRightIcon:I},directives:{ripple:S}},J=["data-p"],K=["aria-label","tabindex"],Z=["data-p"],q=["aria-orientation"],Q=["aria-label","tabindex"];function X(e,t,r,a,s,n){var o=R("ripple");return l(),b("div",i({ref:"list",class:e.cx("root"),"data-p":n.dataP},e.ptmi("root")),[n.showNavigators&&s.isPrevButtonEnabled?g((l(),b("button",i({key:0,ref:"prevButton",type:"button",class:e.cx("prevButton"),"aria-label":n.prevButtonAriaLabel,tabindex:n.$pcTabs.tabindex,onClick:t[0]||(t[0]=function(){return n.onPrevButtonClick&&n.onPrevButtonClick.apply(n,arguments)})},e.ptm("prevButton"),{"data-pc-group-section":"navigator"}),[(l(),B(k(n.templates.previcon||"ChevronLeftIcon"),i({"aria-hidden":"true"},e.ptm("prevIcon")),null,16))],16,K)),[[o]]):m("",!0),f("div",i({ref:"content",class:e.cx("content"),onScroll:t[1]||(t[1]=function(){return n.onScroll&&n.onScroll.apply(n,arguments)}),"data-p":n.dataP},e.ptm("content")),[f("div",i({ref:"tabs",class:e.cx("tabList"),role:"tablist","aria-orientation":n.$pcTabs.orientation||"horizontal"},e.ptm("tabList")),[d(e.$slots,"default"),f("span",i({ref:"inkbar",class:e.cx("activeBar"),role:"presentation","aria-hidden":"true"},e.ptm("activeBar")),null,16)],16,q)],16,Z),n.showNavigators&&s.isNextButtonEnabled?g((l(),b("button",i({key:1,ref:"nextButton",type:"button",class:e.cx("nextButton"),"aria-label":n.nextButtonAriaLabel,tabindex:n.$pcTabs.tabindex,onClick:t[2]||(t[2]=function(){return n.onNextButtonClick&&n.onNextButtonClick.apply(n,arguments)})},e.ptm("nextButton"),{"data-pc-group-section":"navigator"}),[(l(),B(k(n.templates.nexticon||"ChevronRightIcon"),i({"aria-hidden":"true"},e.ptm("nextIcon")),null,16))],16,Q)),[[o]]):m("",!0)],16,J)}G.render=X;var Y=p.extend({name:"tabpanels",classes:{root:"p-tabpanels"}}),tt={name:"TabPanels",extends:{name:"BaseTabPanels",extends:v,props:{},style:Y,provide:function(){return{$pcTabPanels:this,$parentInstance:this}}},inheritAttrs:!1};function et(e,t,r,a,s,n){return l(),b("div",i({class:e.cx("root"),role:"presentation"},e.ptmi("root")),[d(e.$slots,"default")],16)}tt.render=et;var nt=p.extend({name:"tabpanel",classes:{root:function(t){return["p-tabpanel",{"p-tabpanel-active":t.instance.active}]}}}),at={name:"TabPanel",extends:{name:"BaseTabPanel",extends:v,props:{value:{type:[String,Number],default:void 0},as:{type:[String,Object],default:"DIV"},asChild:{type:Boolean,default:!1},header:null,headerStyle:null,headerClass:null,headerProps:null,headerActionProps:null,contentStyle:null,contentClass:null,contentProps:null,disabled:Boolean},style:nt,provide:function(){return{$pcTabPanel:this,$parentInstance:this}}},inheritAttrs:!1,inject:["$pcTabs"],computed:{active:function(){var t;return V((t=this.$pcTabs)===null||t===void 0?void 0:t.d_value,this.value)},id:function(){var t;return"".concat((t=this.$pcTabs)===null||t===void 0?void 0:t.$id,"_tabpanel_").concat(this.value)},ariaLabelledby:function(){var t;return"".concat((t=this.$pcTabs)===null||t===void 0?void 0:t.$id,"_tab_").concat(this.value)},attrs:function(){return i(this.a11yAttrs,this.ptmi("root",this.ptParams))},a11yAttrs:function(){var t;return{id:this.id,tabindex:(t=this.$pcTabs)===null||t===void 0?void 0:t.tabindex,role:"tabpanel","aria-labelledby":this.ariaLabelledby,"data-pc-name":"tabpanel","data-p-active":this.active}},ptParams:function(){return{context:{active:this.active}}}}};function rt(e,t,r,a,s,n){var o,u;return n.$pcTabs?(l(),b(T,{key:1},[e.asChild?d(e.$slots,"default",{key:1,class:_(e.cx("root")),active:n.active,a11yAttrs:n.a11yAttrs}):(l(),b(T,{key:0},[!((o=n.$pcTabs)!==null&&o!==void 0&&o.lazy)||n.active?g((l(),B(k(e.as),i({key:0,class:e.cx("root")},n.attrs),{default:C(function(){return[d(e.$slots,"default")]}),_:3},16,["class"])),[[E,(u=n.$pcTabs)!==null&&u!==void 0&&u.lazy?!0:n.active]]):m("",!0)],64))],64)):d(e.$slots,"default",{key:0})}at.render=rt;export{j as i,tt as n,G as r,at as t};

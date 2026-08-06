import{C as h,Cn as B,Gt as w,Ht as S,Ln as O,Qt as d,R as z,Rn as C,S as P,Sn as N,Xt as p,Yt as T,Zt as y,an as I,ct as m,gn as u,j as f,kt as W,ln as o,mt as g,pn as l,rn as A,vn as R,vt as x,y as _,yn as k,zn as V}from"./vendor-ui-core-DlnFbH2Y.js";import{i as M}from"./vendor-ui-data-CKi5ILYS.js";var E=`
    .p-tabs {
        display: flex;
        flex-direction: column;
    }

    .p-tablist {
        overflow: hidden;
        display: flex;
        position: relative;
        background: dt('tabs.tablist.background');
        border-style: solid;
        border-color: dt('tabs.tablist.border.color');
        border-width: dt('tabs.tablist.border.width');
    }

    .p-tablist-content {
        position: relative;
        display: flex;
        flex-grow: 1;
        min-height: 0;
        overflow-x: auto;
        overflow-y: clip;
        scroll-behavior: smooth;
        scrollbar-width: none;
        overscroll-behavior: contain auto;
    }

    .p-tablist-content::-webkit-scrollbar {
        display: none;
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
        font-size: dt('tabs.tab.font.size');
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
        background: dt('tabs.active.bar.background');
        transition: width 250ms cubic-bezier(0.35, 0, 0.25, 1), inset-inline-start 250ms cubic-bezier(0.35, 0, 0.25, 1);
        inset-inline-start: var(--px-active-bar-left);
        inset-block-end: dt('tabs.active.bar.bottom');
        width: var(--px-active-bar-width);
        height: dt('tabs.active.bar.height');
    }
`,j=f.extend({name:"tabs",style:E,classes:{root:"p-tabs p-component"}}),D={name:"Tabs",extends:{name:"BaseTabs",extends:h,props:{value:{type:[String,Number],default:void 0},lazy:{type:Boolean,default:!1},showNavigators:{type:Boolean,default:!0},tabindex:{type:Number,default:0},selectOnFocus:{type:Boolean,default:!1},scrollable:{type:Boolean,default:!1},scrollStrategy:{type:[String,Function],default:"nearest"}},style:j,provide:function(){return{$pcTabs:this,$parentInstance:this}}},inheritAttrs:!1,emits:["update:value"],data:function(){return{d_value:this.value}},watch:{value:function(t){this.d_value=t}},methods:{updateValue:function(t){this.d_value!==t&&(this.d_value=t,this.$emit("update:value",t))},scrollToActiveTab:function(t,r){if(!(!t||!r||this.scrollStrategy===!1)){if(typeof this.scrollStrategy=="function"){this.scrollStrategy(t,r);return}var a=t.clientWidth,s=Math.abs(t.scrollLeft),n=r.offsetLeft,i=r.offsetWidth,b=n+i,c;if(this.scrollStrategy==="center")c=n-(a-i)/2;else{var v=a*.1;if(n<s+v)c=n-v;else if(b>s+a-v)c=b-a+v;else return}var L=t.scrollWidth-a,$=Math.max(0,Math.min(c,L));t.scrollTo({left:x(t)?-$:$,behavior:"smooth"})}}}};function F(e,t,r,a,s,n){return l(),d("div",o({class:e.cx("root")},e.ptmi("root")),[u(e.$slots,"default")],16)}D.render=F;var H={name:"chevron-left",meta:{tags:["chevron-left","backward","previous","return","left"]},svg:{xmlns:"http://www.w3.org/2000/svg",width:20,height:20,viewBox:"0 0 20 20",fill:"none"},nodes:[["path",{d:"M11.9697 4.46973C12.2626 4.17684 12.7374 4.17684 13.0303 4.46973C13.3232 4.76262 13.3232 5.23738 13.0303 5.53028L8.56055 10L13.0303 14.4697C13.3232 14.7626 13.3232 15.2374 13.0303 15.5303C12.7374 15.8232 12.2626 15.8232 11.9697 15.5303L6.96973 10.5303C6.67684 10.2374 6.67684 9.76262 6.96973 9.46973L11.9697 4.46973Z",fill:"currentColor",key:"es7c15"}]]},U=A({name:"ChevronLeft",inheritAttrs:!1,__name:"chevron-left",setup(e){const{Icon:t}=P(H);return(r,a)=>(l(),p(O(t),V(I(r.$attrs)),null,16))}}),Z=f.extend({name:"tablist",classes:{root:"p-tablist",content:"p-tablist-content",activeBar:"p-tablist-active-bar",prevButton:"p-tablist-prev-button p-tablist-nav-button",nextButton:"p-tablist-next-button p-tablist-nav-button"}}),q={name:"TabList",extends:{name:"BaseTabList",extends:h,props:{},style:Z,provide:function(){return{$pcTabList:this,$parentInstance:this}}},inheritAttrs:!1,inject:["$pcTabs"],data:function(){return{isPrevButtonEnabled:!1,isNextButtonEnabled:!0}},resizeObserver:void 0,inkBarObserver:void 0,mountTimer:null,watch:{showNavigators:function(t){t?this.bindResizeObserver():this.unbindResizeObserver()},activeValue:{flush:"post",handler:function(){this.updateInkBar(),this.bindInkBarObserver();var t=this.$refs.content,r=t?m(t,'[data-pc-name="tab"][data-p-active="true"]'):null;t&&r&&this.$pcTabs.scrollToActiveTab(t,r)}}},mounted:function(){var t=this;this.mountTimer=setTimeout(function(){t.mountTimer=null,t.updateInkBar(),t.bindInkBarObserver()},150),this.showNavigators&&(this.updateButtonState(),this.bindResizeObserver())},updated:function(){this.showNavigators&&this.updateButtonState()},beforeUnmount:function(){this.mountTimer&&(clearTimeout(this.mountTimer),this.mountTimer=null),this.unbindResizeObserver(),this.unbindInkBarObserver()},methods:{onScroll:function(t){this.showNavigators&&this.updateButtonState(),t.preventDefault()},onPrevButtonClick:function(){var t=this.$refs.content,r=this.getVisibleButtonWidths(),a=g(t)-r,s=Math.abs(t.scrollLeft)-a*.8,n=Math.max(s,0);t.scrollLeft=x(t)?-1*n:n},onNextButtonClick:function(){var t=this.$refs.content,r=this.getVisibleButtonWidths(),a=g(t)-r,s=Math.abs(t.scrollLeft)+a*.8,n=t.scrollWidth-a,i=Math.min(s,n);t.scrollLeft=x(t)?-1*i:i},bindResizeObserver:function(){var t=this;this.resizeObserver=new ResizeObserver(function(){return t.updateButtonState()}),this.resizeObserver.observe(this.$refs.list)},unbindResizeObserver:function(){var t;(t=this.resizeObserver)===null||t===void 0||t.unobserve(this.$refs.list),this.resizeObserver=void 0},bindInkBarObserver:function(){var t=this;this.unbindInkBarObserver();var r=this.$refs.content,a=m(r,'[data-pc-name="tab"][data-p-active="true"]');a&&(this.inkBarObserver=new ResizeObserver(function(){return t.updateInkBar()}),this.inkBarObserver.observe(a))},unbindInkBarObserver:function(){var t;(t=this.inkBarObserver)===null||t===void 0||t.disconnect(),this.inkBarObserver=void 0},updateInkBar:function(){var t=this.$refs,r=t.content,a=t.inkbar;if(a){var s=m(r,'[data-pc-name="tab"][data-p-active="true"]');s&&(a.style.setProperty("--px-active-bar-width",s.offsetWidth+"px"),a.style.setProperty("--px-active-bar-height",s.offsetHeight+"px"),a.style.setProperty("--px-active-bar-left",s.offsetLeft+"px"),a.style.setProperty("--px-active-bar-top",s.offsetTop+"px"))}},updateButtonState:function(){var t=this.$refs,r=t.list,a=t.content,s=a.scrollWidth,n=a.offsetWidth,i=Math.abs(a.scrollLeft),b=g(a);this.isPrevButtonEnabled=i!==0,this.isNextButtonEnabled=r.offsetWidth>=n&&parseInt(i)!==s-b},getVisibleButtonWidths:function(){var t=this.$refs,r=t.prevButton,a=t.nextButton,s=0;return this.showNavigators&&(s=(r?.offsetWidth||0)+(a?.offsetWidth||0)),s}},computed:{templates:function(){return this.$pcTabs.$slots},activeValue:function(){return this.$pcTabs.d_value},showNavigators:function(){return this.$pcTabs.showNavigators},prevButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.previous:void 0},nextButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.next:void 0},dataP:function(){return z({scrollable:this.$pcTabs.scrollable})}},components:{ChevronLeft:U,ChevronRight:M},directives:{ripple:_}},G=["data-p"],Q=["aria-label","tabindex"],X=["aria-label","tabindex"];function Y(e,t,r,a,s,n){var i=R("ripple");return l(),d("div",o({ref:"list",class:e.cx("root"),"data-p":n.dataP},e.ptmi("root")),[n.showNavigators&&s.isPrevButtonEnabled?B((l(),d("button",o({key:0,ref:"prevButton",type:"button",class:e.cx("prevButton"),"aria-label":n.prevButtonAriaLabel,tabindex:n.$pcTabs.tabindex,onClick:t[0]||(t[0]=function(){return n.onPrevButtonClick&&n.onPrevButtonClick.apply(n,arguments)})},e.ptm("prevButton"),{"data-pc-group-section":"navigator"}),[(l(),p(k(n.templates.previcon||"ChevronLeft"),o({"aria-hidden":"true"},e.ptm("prevIcon")),null,16))],16,Q)),[[i]]):y("",!0),T("div",o({ref:"content",class:e.cx("content"),role:"tablist","aria-orientation":"horizontal",onScroll:t[1]||(t[1]=function(){return n.onScroll&&n.onScroll.apply(n,arguments)})},e.ptm("content")),[u(e.$slots,"default"),T("span",o({ref:"inkbar",class:e.cx("activeBar"),role:"presentation","aria-hidden":"true"},e.ptm("activeBar")),null,16)],16),n.showNavigators&&s.isNextButtonEnabled?B((l(),d("button",o({key:1,ref:"nextButton",type:"button",class:e.cx("nextButton"),"aria-label":n.nextButtonAriaLabel,tabindex:n.$pcTabs.tabindex,onClick:t[2]||(t[2]=function(){return n.onNextButtonClick&&n.onNextButtonClick.apply(n,arguments)})},e.ptm("nextButton"),{"data-pc-group-section":"navigator"}),[(l(),p(k(n.templates.nexticon||"ChevronRight"),o({"aria-hidden":"true"},e.ptm("nextIcon")),null,16))],16,X)),[[i]]):y("",!0)],16,G)}q.render=Y;var J=f.extend({name:"tabpanels",classes:{root:"p-tabpanels"}}),K={name:"TabPanels",extends:{name:"BaseTabPanels",extends:h,props:{},style:J,provide:function(){return{$pcTabPanels:this,$parentInstance:this}}},inheritAttrs:!1};function tt(e,t,r,a,s,n){return l(),d("div",o({class:e.cx("root"),role:"presentation"},e.ptmi("root")),[u(e.$slots,"default")],16)}K.render=tt;var et=f.extend({name:"tabpanel",classes:{root:function(t){return["p-tabpanel",{"p-tabpanel-active":t.instance.active}]}}}),nt={name:"TabPanel",extends:{name:"BaseTabPanel",extends:h,props:{value:{type:[String,Number],default:void 0},as:{type:[String,Object],default:"DIV"},asChild:{type:Boolean,default:!1}},style:et,provide:function(){return{$pcTabPanel:this,$parentInstance:this}}},inheritAttrs:!1,inject:["$pcTabs"],computed:{active:function(){var t;return W((t=this.$pcTabs)===null||t===void 0?void 0:t.d_value,this.value)},id:function(){var t;return"".concat((t=this.$pcTabs)===null||t===void 0?void 0:t.$id,"_tabpanel_").concat(this.value)},ariaLabelledby:function(){var t;return"".concat((t=this.$pcTabs)===null||t===void 0?void 0:t.$id,"_tab_").concat(this.value)},attrs:function(){return o(this.a11yAttrs,this.ptmi("root",this.ptParams))},a11yAttrs:function(){var t;return{id:this.id,tabindex:(t=this.$pcTabs)===null||t===void 0?void 0:t.tabindex,role:"tabpanel","aria-labelledby":this.ariaLabelledby,"data-pc-name":"tabpanel","data-p-active":this.active}},ptParams:function(){return{context:{active:this.active}}}}};function at(e,t,r,a,s,n){var i,b;return n.$pcTabs?(l(),d(w,{key:1},[e.asChild?u(e.$slots,"default",{key:1,class:C(e.cx("root")),active:n.active,a11yAttrs:n.a11yAttrs}):(l(),d(w,{key:0},[!((i=n.$pcTabs)!==null&&i!==void 0&&i.lazy)||n.active?B((l(),p(k(e.as),o({key:0,class:e.cx("root")},n.attrs),{default:N(function(){return[u(e.$slots,"default")]}),_:3},16,["class"])),[[S,(b=n.$pcTabs)!==null&&b!==void 0&&b.lazy?!0:n.active]]):y("",!0)],64))],64)):u(e.$slots,"default",{key:0})}nt.render=at;export{D as i,K as n,q as r,nt as t};

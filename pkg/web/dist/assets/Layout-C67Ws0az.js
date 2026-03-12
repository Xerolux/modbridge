import{B as q,a9 as M,M as E,ax as J,Y as te,a8 as j,Q as w,aB as ce,aJ as W,x as d,v as S,q as B,o as s,c as m,F as L,g as U,a as h,m as V,s as k,n as O,z as K,d as p,t as R,i as fe,y,h as v,a7 as be,aK as pe,ae as he,p as P,A as ne,K as ve,J as H,S as Ie,aa as N,b as ge,aL as ye,r as D,Z as ke,aM as we,u as I,C as Le,a2 as xe,k as Pe,f as Ce,D as Oe}from"./index-xirL78uc.js";import{x as C,s as ie,F as Se,u as Ke,b as Me}from"./index-1kcWylTa.js";import{a as Ae,s as Ee}from"./index-BxJRJluE.js";import{s as T,f as _}from"./index-DK6hop-p.js";import{s as De}from"./index-R28EC6FF.js";import{R as re,a as ze,s as A}from"./index-B5jrWt4O.js";import{s as Fe}from"./index-ChkC8Lvf.js";import{O as Be,C as Ve}from"./index-CfFuILvQ.js";import{_ as Re}from"./plugin-vue_export-helper-DlAUqK2U.js";import"./index-CeVDVtUA.js";var Te=`
    .p-menubar {
        display: flex;
        align-items: center;
        background: dt('menubar.background');
        border: 1px solid dt('menubar.border.color');
        border-radius: dt('menubar.border.radius');
        color: dt('menubar.color');
        padding: dt('menubar.padding');
        gap: dt('menubar.gap');
    }

    .p-menubar-start,
    .p-megamenu-end {
        display: flex;
        align-items: center;
    }

    .p-menubar-root-list,
    .p-menubar-submenu {
        display: flex;
        margin: 0;
        padding: 0;
        list-style: none;
        outline: 0 none;
    }

    .p-menubar-root-list {
        align-items: center;
        flex-wrap: wrap;
        gap: dt('menubar.gap');
    }

    .p-menubar-root-list > .p-menubar-item > .p-menubar-item-content {
        border-radius: dt('menubar.base.item.border.radius');
    }

    .p-menubar-root-list > .p-menubar-item > .p-menubar-item-content > .p-menubar-item-link {
        padding: dt('menubar.base.item.padding');
    }

    .p-menubar-item-content {
        transition:
            background dt('menubar.transition.duration'),
            color dt('menubar.transition.duration');
        border-radius: dt('menubar.item.border.radius');
        color: dt('menubar.item.color');
    }

    .p-menubar-item-link {
        cursor: pointer;
        display: flex;
        align-items: center;
        text-decoration: none;
        overflow: hidden;
        position: relative;
        color: inherit;
        padding: dt('menubar.item.padding');
        gap: dt('menubar.item.gap');
        user-select: none;
        outline: 0 none;
    }

    .p-menubar-item-label {
        line-height: 1;
    }

    .p-menubar-item-icon {
        color: dt('menubar.item.icon.color');
    }

    .p-menubar-submenu-icon {
        color: dt('menubar.submenu.icon.color');
        margin-left: auto;
        font-size: dt('menubar.submenu.icon.size');
        width: dt('menubar.submenu.icon.size');
        height: dt('menubar.submenu.icon.size');
    }

    .p-menubar-submenu .p-menubar-submenu-icon:dir(rtl) {
        margin-left: 0;
        margin-right: auto;
    }

    .p-menubar-item.p-focus > .p-menubar-item-content {
        color: dt('menubar.item.focus.color');
        background: dt('menubar.item.focus.background');
    }

    .p-menubar-item.p-focus > .p-menubar-item-content .p-menubar-item-icon {
        color: dt('menubar.item.icon.focus.color');
    }

    .p-menubar-item.p-focus > .p-menubar-item-content .p-menubar-submenu-icon {
        color: dt('menubar.submenu.icon.focus.color');
    }

    .p-menubar-item:not(.p-disabled) > .p-menubar-item-content:hover {
        color: dt('menubar.item.focus.color');
        background: dt('menubar.item.focus.background');
    }

    .p-menubar-item:not(.p-disabled) > .p-menubar-item-content:hover .p-menubar-item-icon {
        color: dt('menubar.item.icon.focus.color');
    }

    .p-menubar-item:not(.p-disabled) > .p-menubar-item-content:hover .p-menubar-submenu-icon {
        color: dt('menubar.submenu.icon.focus.color');
    }

    .p-menubar-item-active > .p-menubar-item-content {
        color: dt('menubar.item.active.color');
        background: dt('menubar.item.active.background');
    }

    .p-menubar-item-active > .p-menubar-item-content .p-menubar-item-icon {
        color: dt('menubar.item.icon.active.color');
    }

    .p-menubar-item-active > .p-menubar-item-content .p-menubar-submenu-icon {
        color: dt('menubar.submenu.icon.active.color');
    }

    .p-menubar-submenu {
        display: none;
        position: absolute;
        min-width: 12.5rem;
        z-index: 1;
        background: dt('menubar.submenu.background');
        border: 1px solid dt('menubar.submenu.border.color');
        border-radius: dt('menubar.submenu.border.radius');
        box-shadow: dt('menubar.submenu.shadow');
        color: dt('menubar.submenu.color');
        flex-direction: column;
        padding: dt('menubar.submenu.padding');
        gap: dt('menubar.submenu.gap');
    }

    .p-menubar-submenu .p-menubar-separator {
        border-block-start: 1px solid dt('menubar.separator.border.color');
    }

    .p-menubar-submenu .p-menubar-item {
        position: relative;
    }

    .p-menubar-submenu > .p-menubar-item-active > .p-menubar-submenu {
        display: block;
        left: 100%;
        top: 0;
    }

    .p-menubar-end {
        margin-left: auto;
        align-self: center;
    }

    .p-menubar-end:dir(rtl) {
        margin-left: 0;
        margin-right: auto;
    }

    .p-menubar-button {
        display: none;
        justify-content: center;
        align-items: center;
        cursor: pointer;
        width: dt('menubar.mobile.button.size');
        height: dt('menubar.mobile.button.size');
        position: relative;
        color: dt('menubar.mobile.button.color');
        border: 0 none;
        background: transparent;
        border-radius: dt('menubar.mobile.button.border.radius');
        transition:
            background dt('menubar.transition.duration'),
            color dt('menubar.transition.duration'),
            outline-color dt('menubar.transition.duration');
        outline-color: transparent;
    }

    .p-menubar-button:hover {
        color: dt('menubar.mobile.button.hover.color');
        background: dt('menubar.mobile.button.hover.background');
    }

    .p-menubar-button:focus-visible {
        box-shadow: dt('menubar.mobile.button.focus.ring.shadow');
        outline: dt('menubar.mobile.button.focus.ring.width') dt('menubar.mobile.button.focus.ring.style') dt('menubar.mobile.button.focus.ring.color');
        outline-offset: dt('menubar.mobile.button.focus.ring.offset');
    }

    .p-menubar-mobile {
        position: relative;
    }

    .p-menubar-mobile .p-menubar-button {
        display: flex;
    }

    .p-menubar-mobile .p-menubar-root-list {
        position: absolute;
        display: none;
        width: 100%;
        flex-direction: column;
        top: 100%;
        left: 0;
        z-index: 1;
        padding: dt('menubar.submenu.padding');
        background: dt('menubar.submenu.background');
        border: 1px solid dt('menubar.submenu.border.color');
        box-shadow: dt('menubar.submenu.shadow');
        border-radius: dt('menubar.submenu.border.radius');
        gap: dt('menubar.submenu.gap');
    }

    .p-menubar-mobile .p-menubar-root-list:dir(rtl) {
        left: auto;
        right: 0;
    }

    .p-menubar-mobile .p-menubar-root-list > .p-menubar-item > .p-menubar-item-content > .p-menubar-item-link {
        padding: dt('menubar.item.padding');
    }

    .p-menubar-mobile-active .p-menubar-root-list {
        display: flex;
    }

    .p-menubar-mobile .p-menubar-root-list .p-menubar-item {
        width: 100%;
        position: static;
    }

    .p-menubar-mobile .p-menubar-root-list .p-menubar-separator {
        border-block-start: 1px solid dt('menubar.separator.border.color');
    }

    .p-menubar-mobile .p-menubar-root-list > .p-menubar-item > .p-menubar-item-content .p-menubar-submenu-icon {
        margin-left: auto;
        transition: transform 0.2s;
    }

    .p-menubar-mobile .p-menubar-root-list > .p-menubar-item > .p-menubar-item-content .p-menubar-submenu-icon:dir(rtl),
    .p-menubar-mobile .p-menubar-submenu-icon:dir(rtl) {
        margin-left: 0;
        margin-right: auto;
    }

    .p-menubar-mobile .p-menubar-root-list > .p-menubar-item-active > .p-menubar-item-content .p-menubar-submenu-icon {
        transform: rotate(-180deg);
    }

    .p-menubar-mobile .p-menubar-submenu .p-menubar-submenu-icon {
        transition: transform 0.2s;
        transform: rotate(90deg);
    }

    .p-menubar-mobile .p-menubar-item-active > .p-menubar-item-content .p-menubar-submenu-icon {
        transform: rotate(-90deg);
    }

    .p-menubar-mobile .p-menubar-submenu {
        width: 100%;
        position: static;
        box-shadow: none;
        border: 0 none;
        padding-inline-start: dt('menubar.submenu.mobile.indent');
        padding-inline-end: 0;
    }
`,je={submenu:function(e){var n=e.instance,r=e.processedItem;return{display:n.isItemActive(r)?"flex":"none"}}},Ne={root:function(e){var n=e.instance;return["p-menubar p-component",{"p-menubar-mobile":n.queryMatches,"p-menubar-mobile-active":n.mobileActive}]},start:"p-menubar-start",button:"p-menubar-button",rootList:"p-menubar-root-list",item:function(e){var n=e.instance,r=e.processedItem;return["p-menubar-item",{"p-menubar-item-active":n.isItemActive(r),"p-focus":n.isItemFocused(r),"p-disabled":n.isItemDisabled(r)}]},itemContent:"p-menubar-item-content",itemLink:"p-menubar-item-link",itemIcon:"p-menubar-item-icon",itemLabel:"p-menubar-item-label",submenuIcon:"p-menubar-submenu-icon",submenu:"p-menubar-submenu",separator:"p-menubar-separator",end:"p-menubar-end"},Ue=q.extend({name:"menubar",style:Te,classes:Ne,inlineStyles:je}),He={name:"BaseMenubar",extends:T,props:{model:{type:Array,default:null},buttonProps:{type:null,default:null},breakpoint:{type:String,default:"960px"},ariaLabelledby:{type:String,default:null},ariaLabel:{type:String,default:null}},style:Ue,provide:function(){return{$pcMenubar:this,$parentInstance:this}}},oe={name:"MenubarSub",hostName:"Menubar",extends:T,emits:["item-mouseenter","item-click","item-mousemove"],props:{items:{type:Array,default:null},root:{type:Boolean,default:!1},popup:{type:Boolean,default:!1},mobileActive:{type:Boolean,default:!1},templates:{type:Object,default:null},level:{type:Number,default:0},menuId:{type:String,default:null},focusedItemId:{type:String,default:null},activeItemPath:{type:Object,default:null}},list:null,methods:{getItemId:function(e){return"".concat(this.menuId,"_").concat(e.key)},getItemKey:function(e){return this.getItemId(e)},getItemProp:function(e,n,r){return e&&e.item?W(e.item[n],r):void 0},getItemLabel:function(e){return this.getItemProp(e,"label")},getItemLabelId:function(e){return"".concat(this.menuId,"_").concat(e.key,"_label")},getPTOptions:function(e,n,r){return this.ptm(r,{context:{item:e.item,index:n,active:this.isItemActive(e),focused:this.isItemFocused(e),disabled:this.isItemDisabled(e),level:this.level}})},isItemActive:function(e){return this.activeItemPath.some(function(n){return n.key===e.key})},isItemVisible:function(e){return this.getItemProp(e,"visible")!==!1},isItemDisabled:function(e){return this.getItemProp(e,"disabled")},isItemFocused:function(e){return this.focusedItemId===this.getItemId(e)},isItemGroup:function(e){return M(e.items)},onItemClick:function(e,n){this.getItemProp(n,"command",{originalEvent:e,item:n.item}),this.$emit("item-click",{originalEvent:e,processedItem:n,isFocus:!0})},onItemMouseEnter:function(e,n){this.$emit("item-mouseenter",{originalEvent:e,processedItem:n})},onItemMouseMove:function(e,n){this.$emit("item-mousemove",{originalEvent:e,processedItem:n})},getAriaPosInset:function(e){return e-this.calculateAriaSetSize.slice(0,e).length+1},getMenuItemProps:function(e,n){return{action:d({class:this.cx("itemLink"),tabindex:-1},this.getPTOptions(e,n,"itemLink")),icon:d({class:[this.cx("itemIcon"),this.getItemProp(e,"icon")]},this.getPTOptions(e,n,"itemIcon")),label:d({class:this.cx("itemLabel")},this.getPTOptions(e,n,"itemLabel")),submenuicon:d({class:this.cx("submenuIcon")},this.getPTOptions(e,n,"submenuIcon"))}}},computed:{calculateAriaSetSize:function(){var e=this;return this.items.filter(function(n){return e.isItemVisible(n)&&e.getItemProp(n,"separator")})},getAriaSetSize:function(){var e=this;return this.items.filter(function(n){return e.isItemVisible(n)&&!e.getItemProp(n,"separator")}).length}},components:{AngleRightIcon:Ee,AngleDownIcon:De},directives:{ripple:re}},Ge=["id","aria-label","aria-disabled","aria-expanded","aria-haspopup","aria-setsize","aria-posinset","data-p-active","data-p-focused","data-p-disabled"],Ze=["onClick","onMouseenter","onMousemove"],qe=["href","target"],We=["id"],_e=["id"];function Je(t,e,n,r,a,i){var u=S("MenubarSub",!0),c=B("ripple");return s(),m("ul",d({class:n.level===0?t.cx("rootList"):t.cx("submenu")},n.level===0?t.ptm("rootList"):t.ptm("submenu")),[(s(!0),m(L,null,U(n.items,function(o,l){return s(),m(L,{key:i.getItemKey(o)},[i.isItemVisible(o)&&!i.getItemProp(o,"separator")?(s(),m("li",d({key:0,id:i.getItemId(o),style:i.getItemProp(o,"style"),class:[t.cx("item",{processedItem:o}),i.getItemProp(o,"class")],role:"menuitem","aria-label":i.getItemLabel(o),"aria-disabled":i.isItemDisabled(o)||void 0,"aria-expanded":i.isItemGroup(o)?i.isItemActive(o):void 0,"aria-haspopup":i.isItemGroup(o)&&!i.getItemProp(o,"to")?"menu":void 0,"aria-setsize":i.getAriaSetSize,"aria-posinset":i.getAriaPosInset(l)},{ref_for:!0},i.getPTOptions(o,l,"item"),{"data-p-active":i.isItemActive(o),"data-p-focused":i.isItemFocused(o),"data-p-disabled":i.isItemDisabled(o)}),[h("div",d({class:t.cx("itemContent"),onClick:function(b){return i.onItemClick(b,o)},onMouseenter:function(b){return i.onItemMouseEnter(b,o)},onMousemove:function(b){return i.onItemMouseMove(b,o)}},{ref_for:!0},i.getPTOptions(o,l,"itemContent")),[n.templates.item?(s(),k(K(n.templates.item),{key:1,item:o.item,root:n.root,hasSubmenu:i.getItemProp(o,"items"),label:i.getItemLabel(o),props:i.getMenuItemProps(o,l)},null,8,["item","root","hasSubmenu","label","props"])):V((s(),m("a",d({key:0,href:i.getItemProp(o,"url"),class:t.cx("itemLink"),target:i.getItemProp(o,"target"),tabindex:"-1"},{ref_for:!0},i.getPTOptions(o,l,"itemLink")),[n.templates.itemicon?(s(),k(K(n.templates.itemicon),{key:0,item:o.item,class:O(t.cx("itemIcon"))},null,8,["item","class"])):i.getItemProp(o,"icon")?(s(),m("span",d({key:1,class:[t.cx("itemIcon"),i.getItemProp(o,"icon")]},{ref_for:!0},i.getPTOptions(o,l,"itemIcon")),null,16)):p("",!0),h("span",d({id:i.getItemLabelId(o),class:t.cx("itemLabel")},{ref_for:!0},i.getPTOptions(o,l,"itemLabel")),R(i.getItemLabel(o)),17,We),i.getItemProp(o,"items")?(s(),m(L,{key:2},[n.templates.submenuicon?(s(),k(K(n.templates.submenuicon),{key:0,root:n.root,active:i.isItemActive(o),class:O(t.cx("submenuIcon"))},null,8,["root","active","class"])):(s(),k(K(n.root?"AngleDownIcon":"AngleRightIcon"),d({key:1,class:t.cx("submenuIcon")},{ref_for:!0},i.getPTOptions(o,l,"submenuIcon")),null,16,["class"]))],64)):p("",!0)],16,qe)),[[c]])],16,Ze),i.isItemVisible(o)&&i.isItemGroup(o)?(s(),k(u,{key:0,id:i.getItemId(o)+"_list",menuId:n.menuId,role:"menu",style:fe(t.sx("submenu",!0,{processedItem:o})),focusedItemId:n.focusedItemId,items:o.items,mobileActive:n.mobileActive,activeItemPath:n.activeItemPath,templates:n.templates,level:n.level+1,"aria-labelledby":i.getItemLabelId(o),pt:t.pt,unstyled:t.unstyled,onItemClick:e[0]||(e[0]=function(f){return t.$emit("item-click",f)}),onItemMouseenter:e[1]||(e[1]=function(f){return t.$emit("item-mouseenter",f)}),onItemMousemove:e[2]||(e[2]=function(f){return t.$emit("item-mousemove",f)})},null,8,["id","menuId","style","focusedItemId","items","mobileActive","activeItemPath","templates","level","aria-labelledby","pt","unstyled"])):p("",!0)],16,Ge)):p("",!0),i.isItemVisible(o)&&i.getItemProp(o,"separator")?(s(),m("li",d({key:1,id:i.getItemId(o),class:[t.cx("separator"),i.getItemProp(o,"class")],style:i.getItemProp(o,"style"),role:"separator"},{ref_for:!0},t.ptm("separator")),null,16,_e)):p("",!0)],64)}),128))],16)}oe.render=Je;var ae={name:"Menubar",extends:He,inheritAttrs:!1,emits:["focus","blur"],matchMediaListener:null,data:function(){return{mobileActive:!1,focused:!1,focusedItemInfo:{index:-1,level:0,parentKey:""},activeItemPath:[],dirty:!1,query:null,queryMatches:!1}},watch:{activeItemPath:function(e){M(e)?(this.bindOutsideClickListener(),this.bindResizeListener()):(this.unbindOutsideClickListener(),this.unbindResizeListener())}},outsideClickListener:null,container:null,menubar:null,mounted:function(){this.bindMatchMediaListener()},beforeUnmount:function(){this.mobileActive=!1,this.unbindOutsideClickListener(),this.unbindResizeListener(),this.unbindMatchMediaListener(),this.container&&C.clear(this.container),this.container=null},methods:{getItemProp:function(e,n){return e?W(e[n]):void 0},getItemLabel:function(e){return this.getItemProp(e,"label")},isItemDisabled:function(e){return this.getItemProp(e,"disabled")},isItemVisible:function(e){return this.getItemProp(e,"visible")!==!1},isItemGroup:function(e){return M(this.getItemProp(e,"items"))},isItemSeparator:function(e){return this.getItemProp(e,"separator")},getProccessedItemLabel:function(e){return e?this.getItemLabel(e.item):void 0},isProccessedItemGroup:function(e){return e&&M(e.items)},toggle:function(e){var n=this;this.mobileActive?(this.mobileActive=!1,C.clear(this.menubar),this.hide()):(this.mobileActive=!0,C.set("menu",this.menubar,this.$primevue.config.zIndex.menu),setTimeout(function(){n.show()},1)),this.bindOutsideClickListener(),e.preventDefault()},show:function(){w(this.menubar)},hide:function(e,n){var r=this;this.mobileActive&&(this.mobileActive=!1,setTimeout(function(){w(r.$refs.menubutton)},0)),this.activeItemPath=[],this.focusedItemInfo={index:-1,level:0,parentKey:""},n&&w(this.menubar),this.dirty=!1},onFocus:function(e){this.focused=!0,this.focusedItemInfo=this.focusedItemInfo.index!==-1?this.focusedItemInfo:{index:this.findFirstFocusedItemIndex(),level:0,parentKey:""},this.$emit("focus",e)},onBlur:function(e){this.focused=!1,this.focusedItemInfo={index:-1,level:0,parentKey:""},this.searchValue="",this.dirty=!1,this.$emit("blur",e)},onKeyDown:function(e){var n=e.metaKey||e.ctrlKey;switch(e.code){case"ArrowDown":this.onArrowDownKey(e);break;case"ArrowUp":this.onArrowUpKey(e);break;case"ArrowLeft":this.onArrowLeftKey(e);break;case"ArrowRight":this.onArrowRightKey(e);break;case"Home":this.onHomeKey(e);break;case"End":this.onEndKey(e);break;case"Space":this.onSpaceKey(e);break;case"Enter":case"NumpadEnter":this.onEnterKey(e);break;case"Escape":this.onEscapeKey(e);break;case"Tab":this.onTabKey(e);break;case"PageDown":case"PageUp":case"Backspace":case"ShiftLeft":case"ShiftRight":break;default:!n&&ce(e.key)&&this.searchItems(e,e.key);break}},onItemChange:function(e,n){var r=e.processedItem,a=e.isFocus;if(!j(r)){var i=r.index,u=r.key,c=r.level,o=r.parentKey,l=r.items,f=M(l),b=this.activeItemPath.filter(function(x){return x.parentKey!==o&&x.parentKey!==u});f&&b.push(r),this.focusedItemInfo={index:i,level:c,parentKey:o},f&&(this.dirty=!0),a&&w(this.menubar),!(n==="hover"&&this.queryMatches)&&(this.activeItemPath=b)}},onItemClick:function(e){var n=e.originalEvent,r=e.processedItem,a=this.isProccessedItemGroup(r),i=j(r.parent),u=this.isSelected(r);if(u){var c=r.index,o=r.key,l=r.level,f=r.parentKey;this.activeItemPath=this.activeItemPath.filter(function(x){return o!==x.key&&o.startsWith(x.key)}),this.focusedItemInfo={index:c,level:l,parentKey:f},this.dirty=!i,w(this.menubar)}else if(a)this.onItemChange(e);else{var b=i?r:this.activeItemPath.find(function(x){return x.parentKey===""});this.hide(n),this.changeFocusedItemIndex(n,b?b.index:-1),this.mobileActive=!1,w(this.menubar)}},onItemMouseEnter:function(e){this.dirty&&this.onItemChange(e,"hover")},onItemMouseMove:function(e){this.focused&&this.changeFocusedItemIndex(e,e.processedItem.index)},menuButtonClick:function(e){this.toggle(e)},menuButtonKeydown:function(e){(e.code==="Enter"||e.code==="NumpadEnter"||e.code==="Space")&&this.menuButtonClick(e)},onArrowDownKey:function(e){var n=this.visibleItems[this.focusedItemInfo.index],r=n?j(n.parent):null;if(r){var a=this.isProccessedItemGroup(n);a&&(this.onItemChange({originalEvent:e,processedItem:n}),this.focusedItemInfo={index:-1,parentKey:n.key},this.onArrowRightKey(e))}else{var i=this.focusedItemInfo.index!==-1?this.findNextItemIndex(this.focusedItemInfo.index):this.findFirstFocusedItemIndex();this.changeFocusedItemIndex(e,i)}e.preventDefault()},onArrowUpKey:function(e){var n=this,r=this.visibleItems[this.focusedItemInfo.index],a=j(r.parent);if(a){var i=this.isProccessedItemGroup(r);if(i){this.onItemChange({originalEvent:e,processedItem:r}),this.focusedItemInfo={index:-1,parentKey:r.key};var u=this.findLastItemIndex();this.changeFocusedItemIndex(e,u)}}else{var c=this.activeItemPath.find(function(l){return l.key===r.parentKey});if(this.focusedItemInfo.index===0)this.focusedItemInfo={index:-1,parentKey:c?c.parentKey:""},this.searchValue="",this.onArrowLeftKey(e),this.activeItemPath=this.activeItemPath.filter(function(l){return l.parentKey!==n.focusedItemInfo.parentKey});else{var o=this.focusedItemInfo.index!==-1?this.findPrevItemIndex(this.focusedItemInfo.index):this.findLastFocusedItemIndex();this.changeFocusedItemIndex(e,o)}}e.preventDefault()},onArrowLeftKey:function(e){var n=this,r=this.visibleItems[this.focusedItemInfo.index],a=r?this.activeItemPath.find(function(u){return u.key===r.parentKey}):null;if(a)this.onItemChange({originalEvent:e,processedItem:a}),this.activeItemPath=this.activeItemPath.filter(function(u){return u.parentKey!==n.focusedItemInfo.parentKey}),e.preventDefault();else{var i=this.focusedItemInfo.index!==-1?this.findPrevItemIndex(this.focusedItemInfo.index):this.findLastFocusedItemIndex();this.changeFocusedItemIndex(e,i),e.preventDefault()}},onArrowRightKey:function(e){var n=this.visibleItems[this.focusedItemInfo.index],r=n?this.activeItemPath.find(function(u){return u.key===n.parentKey}):null;if(r){var a=this.isProccessedItemGroup(n);a&&(this.onItemChange({originalEvent:e,processedItem:n}),this.focusedItemInfo={index:-1,parentKey:n.key},this.onArrowDownKey(e))}else{var i=this.focusedItemInfo.index!==-1?this.findNextItemIndex(this.focusedItemInfo.index):this.findFirstFocusedItemIndex();this.changeFocusedItemIndex(e,i),e.preventDefault()}},onHomeKey:function(e){this.changeFocusedItemIndex(e,this.findFirstItemIndex()),e.preventDefault()},onEndKey:function(e){this.changeFocusedItemIndex(e,this.findLastItemIndex()),e.preventDefault()},onEnterKey:function(e){if(this.focusedItemInfo.index!==-1){var n=E(this.menubar,'li[id="'.concat("".concat(this.focusedItemId),'"]')),r=n&&E(n,'a[data-pc-section="itemlink"]');r?r.click():n&&n.click();var a=this.visibleItems[this.focusedItemInfo.index],i=this.isProccessedItemGroup(a);!i&&(this.focusedItemInfo.index=this.findFirstFocusedItemIndex())}e.preventDefault()},onSpaceKey:function(e){this.onEnterKey(e)},onEscapeKey:function(e){if(this.focusedItemInfo.level!==0){var n=this.focusedItemInfo;this.hide(e,!1),this.focusedItemInfo={index:Number(n.parentKey.split("_")[0]),level:0,parentKey:""}}e.preventDefault()},onTabKey:function(e){if(this.focusedItemInfo.index!==-1){var n=this.visibleItems[this.focusedItemInfo.index],r=this.isProccessedItemGroup(n);!r&&this.onItemChange({originalEvent:e,processedItem:n})}this.hide()},bindOutsideClickListener:function(){var e=this;this.outsideClickListener||(this.outsideClickListener=function(n){var r=e.container&&!e.container.contains(n.target),a=!(e.target&&(e.target===n.target||e.target.contains(n.target)));r&&a&&e.hide()},document.addEventListener("click",this.outsideClickListener,!0))},unbindOutsideClickListener:function(){this.outsideClickListener&&(document.removeEventListener("click",this.outsideClickListener,!0),this.outsideClickListener=null)},bindResizeListener:function(){var e=this;this.resizeListener||(this.resizeListener=function(n){te()||e.hide(n,!0),e.mobileActive=!1},window.addEventListener("resize",this.resizeListener))},unbindResizeListener:function(){this.resizeListener&&(window.removeEventListener("resize",this.resizeListener),this.resizeListener=null)},bindMatchMediaListener:function(){var e=this;if(!this.matchMediaListener){var n=matchMedia("(max-width: ".concat(this.breakpoint,")"));this.query=n,this.queryMatches=n.matches,this.matchMediaListener=function(){e.queryMatches=n.matches,e.mobileActive=!1},this.query.addEventListener("change",this.matchMediaListener)}},unbindMatchMediaListener:function(){this.matchMediaListener&&(this.query.removeEventListener("change",this.matchMediaListener),this.matchMediaListener=null)},isItemMatched:function(e){var n;return this.isValidItem(e)&&((n=this.getProccessedItemLabel(e))===null||n===void 0?void 0:n.toLocaleLowerCase().startsWith(this.searchValue.toLocaleLowerCase()))},isValidItem:function(e){return!!e&&!this.isItemDisabled(e.item)&&!this.isItemSeparator(e.item)&&this.isItemVisible(e.item)},isValidSelectedItem:function(e){return this.isValidItem(e)&&this.isSelected(e)},isSelected:function(e){return this.activeItemPath.some(function(n){return n.key===e.key})},findFirstItemIndex:function(){var e=this;return this.visibleItems.findIndex(function(n){return e.isValidItem(n)})},findLastItemIndex:function(){var e=this;return J(this.visibleItems,function(n){return e.isValidItem(n)})},findNextItemIndex:function(e){var n=this,r=e<this.visibleItems.length-1?this.visibleItems.slice(e+1).findIndex(function(a){return n.isValidItem(a)}):-1;return r>-1?r+e+1:e},findPrevItemIndex:function(e){var n=this,r=e>0?J(this.visibleItems.slice(0,e),function(a){return n.isValidItem(a)}):-1;return r>-1?r:e},findSelectedItemIndex:function(){var e=this;return this.visibleItems.findIndex(function(n){return e.isValidSelectedItem(n)})},findFirstFocusedItemIndex:function(){var e=this.findSelectedItemIndex();return e<0?this.findFirstItemIndex():e},findLastFocusedItemIndex:function(){var e=this.findSelectedItemIndex();return e<0?this.findLastItemIndex():e},searchItems:function(e,n){var r=this;this.searchValue=(this.searchValue||"")+n;var a=-1,i=!1;return this.focusedItemInfo.index!==-1?(a=this.visibleItems.slice(this.focusedItemInfo.index).findIndex(function(u){return r.isItemMatched(u)}),a=a===-1?this.visibleItems.slice(0,this.focusedItemInfo.index).findIndex(function(u){return r.isItemMatched(u)}):a+this.focusedItemInfo.index):a=this.visibleItems.findIndex(function(u){return r.isItemMatched(u)}),a!==-1&&(i=!0),a===-1&&this.focusedItemInfo.index===-1&&(a=this.findFirstFocusedItemIndex()),a!==-1&&this.changeFocusedItemIndex(e,a),this.searchTimeout&&clearTimeout(this.searchTimeout),this.searchTimeout=setTimeout(function(){r.searchValue="",r.searchTimeout=null},500),i},changeFocusedItemIndex:function(e,n){this.focusedItemInfo.index!==n&&(this.focusedItemInfo.index=n,this.scrollInView())},scrollInView:function(){var e=arguments.length>0&&arguments[0]!==void 0?arguments[0]:-1,n=e!==-1?"".concat(this.$id,"_").concat(e):this.focusedItemId,r=E(this.menubar,'li[id="'.concat(n,'"]'));r&&r.scrollIntoView&&r.scrollIntoView({block:"nearest",inline:"start"})},createProcessedItems:function(e){var n=this,r=arguments.length>1&&arguments[1]!==void 0?arguments[1]:0,a=arguments.length>2&&arguments[2]!==void 0?arguments[2]:{},i=arguments.length>3&&arguments[3]!==void 0?arguments[3]:"",u=[];return e&&e.forEach(function(c,o){var l=(i!==""?i+"_":"")+o,f={item:c,index:o,level:r,key:l,parent:a,parentKey:i};f.items=n.createProcessedItems(c.items,r+1,f,l),u.push(f)}),u},containerRef:function(e){this.container=e},menubarRef:function(e){this.menubar=e?e.$el:void 0}},computed:{processedItems:function(){return this.createProcessedItems(this.model||[])},visibleItems:function(){var e=this,n=this.activeItemPath.find(function(r){return r.key===e.focusedItemInfo.parentKey});return n?n.items:this.processedItems},focusedItemId:function(){return this.focusedItemInfo.index!==-1?"".concat(this.$id).concat(M(this.focusedItemInfo.parentKey)?"_"+this.focusedItemInfo.parentKey:"","_").concat(this.focusedItemInfo.index):null}},components:{MenubarSub:oe,BarsIcon:Ae}};function z(t){"@babel/helpers - typeof";return z=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(e){return typeof e}:function(e){return e&&typeof Symbol=="function"&&e.constructor===Symbol&&e!==Symbol.prototype?"symbol":typeof e},z(t)}function Y(t,e){var n=Object.keys(t);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(t);e&&(r=r.filter(function(a){return Object.getOwnPropertyDescriptor(t,a).enumerable})),n.push.apply(n,r)}return n}function Q(t){for(var e=1;e<arguments.length;e++){var n=arguments[e]!=null?arguments[e]:{};e%2?Y(Object(n),!0).forEach(function(r){Ye(t,r,n[r])}):Object.getOwnPropertyDescriptors?Object.defineProperties(t,Object.getOwnPropertyDescriptors(n)):Y(Object(n)).forEach(function(r){Object.defineProperty(t,r,Object.getOwnPropertyDescriptor(n,r))})}return t}function Ye(t,e,n){return(e=Qe(e))in t?Object.defineProperty(t,e,{value:n,enumerable:!0,configurable:!0,writable:!0}):t[e]=n,t}function Qe(t){var e=Xe(t,"string");return z(e)=="symbol"?e:e+""}function Xe(t,e){if(z(t)!="object"||!t)return t;var n=t[Symbol.toPrimitive];if(n!==void 0){var r=n.call(t,e);if(z(r)!="object")return r;throw new TypeError("@@toPrimitive must return a primitive value.")}return(e==="string"?String:Number)(t)}var $e=["aria-haspopup","aria-expanded","aria-controls","aria-label"];function et(t,e,n,r,a,i){var u=S("BarsIcon"),c=S("MenubarSub");return s(),m("div",d({ref:i.containerRef,class:t.cx("root")},t.ptmi("root")),[t.$slots.start?(s(),m("div",d({key:0,class:t.cx("start")},t.ptm("start")),[y(t.$slots,"start")],16)):p("",!0),y(t.$slots,t.$slots.button?"button":"menubutton",{id:t.$id,class:O(t.cx("button")),toggleCallback:function(l){return i.menuButtonClick(l)}},function(){var o;return[t.model&&t.model.length>0?(s(),m("a",d({key:0,ref:"menubutton",role:"button",tabindex:"0",class:t.cx("button"),"aria-haspopup":!!(t.model.length&&t.model.length>0),"aria-expanded":a.mobileActive,"aria-controls":t.$id,"aria-label":(o=t.$primevue.config.locale.aria)===null||o===void 0?void 0:o.navigation,onClick:e[0]||(e[0]=function(l){return i.menuButtonClick(l)}),onKeydown:e[1]||(e[1]=function(l){return i.menuButtonKeydown(l)})},Q(Q({},t.buttonProps),t.ptm("button"))),[y(t.$slots,t.$slots.buttonicon?"buttonicon":"menubuttonicon",{},function(){return[v(u,be(pe(t.ptm("buttonicon"))),null,16)]})],16,$e)):p("",!0)]}),v(c,{ref:i.menubarRef,id:t.$id+"_list",role:"menubar",items:i.processedItems,templates:t.$slots,root:!0,mobileActive:a.mobileActive,tabindex:"0","aria-activedescendant":a.focused?i.focusedItemId:void 0,menuId:t.$id,focusedItemId:a.focused?i.focusedItemId:void 0,activeItemPath:a.activeItemPath,level:0,"aria-labelledby":t.ariaLabelledby,"aria-label":t.ariaLabel,pt:t.pt,unstyled:t.unstyled,onFocus:i.onFocus,onBlur:i.onBlur,onKeydown:i.onKeyDown,onItemClick:i.onItemClick,onItemMouseenter:i.onItemMouseEnter,onItemMousemove:i.onItemMouseMove},null,8,["id","items","templates","mobileActive","aria-activedescendant","menuId","focusedItemId","activeItemPath","aria-labelledby","aria-label","pt","unstyled","onFocus","onBlur","onKeydown","onItemClick","onItemMouseenter","onItemMousemove"]),t.$slots.end?(s(),m("div",d({key:1,class:t.cx("end")},t.ptm("end")),[y(t.$slots,"end")],16)):p("",!0)],16)}ae.render=et;var tt=`
    .p-drawer {
        display: flex;
        flex-direction: column;
        transform: translate3d(0px, 0px, 0px);
        position: relative;
        transition: transform 0.3s;
        background: dt('drawer.background');
        color: dt('drawer.color');
        border-style: solid;
        border-color: dt('drawer.border.color');
        box-shadow: dt('drawer.shadow');
    }

    .p-drawer-content {
        overflow-y: auto;
        flex-grow: 1;
        padding: dt('drawer.content.padding');
    }

    .p-drawer-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-shrink: 0;
        padding: dt('drawer.header.padding');
    }

    .p-drawer-footer {
        padding: dt('drawer.footer.padding');
    }

    .p-drawer-title {
        font-weight: dt('drawer.title.font.weight');
        font-size: dt('drawer.title.font.size');
    }

    .p-drawer-full .p-drawer {
        transition: none;
        transform: none;
        width: 100vw !important;
        height: 100vh !important;
        max-height: 100%;
        top: 0px !important;
        left: 0px !important;
        border-width: 1px;
    }

    .p-drawer-left .p-drawer-enter-active {
        animation: p-animate-drawer-enter-left 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    .p-drawer-left .p-drawer-leave-active {
        animation: p-animate-drawer-leave-left 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }

    .p-drawer-right .p-drawer-enter-active {
        animation: p-animate-drawer-enter-right 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    .p-drawer-right .p-drawer-leave-active {
        animation: p-animate-drawer-leave-right 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }

    .p-drawer-top .p-drawer-enter-active {
        animation: p-animate-drawer-enter-top 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    .p-drawer-top .p-drawer-leave-active {
        animation: p-animate-drawer-leave-top 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }

    .p-drawer-bottom .p-drawer-enter-active {
        animation: p-animate-drawer-enter-bottom 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    .p-drawer-bottom .p-drawer-leave-active {
        animation: p-animate-drawer-leave-bottom 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }

    .p-drawer-full .p-drawer-enter-active {
        animation: p-animate-drawer-enter-full 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    .p-drawer-full .p-drawer-leave-active {
        animation: p-animate-drawer-leave-full 0.5s cubic-bezier(0.32, 0.72, 0, 1);
    }
    
    .p-drawer-left .p-drawer {
        width: 20rem;
        height: 100%;
        border-inline-end-width: 1px;
    }

    .p-drawer-right .p-drawer {
        width: 20rem;
        height: 100%;
        border-inline-start-width: 1px;
    }

    .p-drawer-top .p-drawer {
        height: 10rem;
        width: 100%;
        border-block-end-width: 1px;
    }

    .p-drawer-bottom .p-drawer {
        height: 10rem;
        width: 100%;
        border-block-start-width: 1px;
    }

    .p-drawer-left .p-drawer-content,
    .p-drawer-right .p-drawer-content,
    .p-drawer-top .p-drawer-content,
    .p-drawer-bottom .p-drawer-content {
        width: 100%;
        height: 100%;
    }

    .p-drawer-open {
        display: flex;
    }

    .p-drawer-mask:dir(rtl) {
        flex-direction: row-reverse;
    }

    @keyframes p-animate-drawer-enter-left {
        from {
            transform: translate3d(-100%, 0px, 0px);
        }
    }

    @keyframes p-animate-drawer-leave-left {
        to {
            transform: translate3d(-100%, 0px, 0px);
        }
    }

    @keyframes p-animate-drawer-enter-right {
        from {
            transform: translate3d(100%, 0px, 0px);
        }
    }

    @keyframes p-animate-drawer-leave-right {
        to {
            transform: translate3d(100%, 0px, 0px);
        }
    }

    @keyframes p-animate-drawer-enter-top {
        from {
            transform: translate3d(0px, -100%, 0px);
        }
    }

    @keyframes p-animate-drawer-leave-top {
        to {
            transform: translate3d(0px, -100%, 0px);
        }
    }

    @keyframes p-animate-drawer-enter-bottom {
        from {
            transform: translate3d(0px, 100%, 0px);
        }
    }

    @keyframes p-animate-drawer-leave-bottom {
        to {
            transform: translate3d(0px, 100%, 0px);
        }
    }

    @keyframes p-animate-drawer-enter-full {
        from {
            opacity: 0;
            transform: scale(0.93);
        }
    }

    @keyframes p-animate-drawer-leave-full {
        to {
            opacity: 0;
            transform: scale(0.93);
        }
    }
`,nt={mask:function(e){var n=e.position,r=e.modal;return{position:"fixed",height:"100%",width:"100%",left:0,top:0,display:"flex",justifyContent:n==="left"?"flex-start":n==="right"?"flex-end":"center",alignItems:n==="top"?"flex-start":n==="bottom"?"flex-end":"center",pointerEvents:r?"auto":"none"}},root:{pointerEvents:"auto"}},it={mask:function(e){var n=e.instance,r=e.props,a=["left","right","top","bottom"],i=a.find(function(u){return u===r.position});return["p-drawer-mask",{"p-overlay-mask p-overlay-mask-enter-active":r.modal,"p-drawer-open":n.containerVisible,"p-drawer-full":n.fullScreen},i?"p-drawer-".concat(i):""]},root:function(e){var n=e.instance;return["p-drawer p-component",{"p-drawer-full":n.fullScreen}]},header:"p-drawer-header",title:"p-drawer-title",pcCloseButton:"p-drawer-close-button",content:"p-drawer-content",footer:"p-drawer-footer"},rt=q.extend({name:"drawer",style:tt,classes:it,inlineStyles:nt}),ot={name:"BaseDrawer",extends:T,props:{visible:{type:Boolean,default:!1},position:{type:String,default:"left"},header:{type:null,default:null},baseZIndex:{type:Number,default:0},autoZIndex:{type:Boolean,default:!0},dismissable:{type:Boolean,default:!0},showCloseIcon:{type:Boolean,default:!0},closeButtonProps:{type:Object,default:function(){return{severity:"secondary",text:!0,rounded:!0}}},closeIcon:{type:String,default:void 0},modal:{type:Boolean,default:!0},blockScroll:{type:Boolean,default:!1},closeOnEscape:{type:Boolean,default:!0}},style:rt,provide:function(){return{$pcDrawer:this,$parentInstance:this}}};function F(t){"@babel/helpers - typeof";return F=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(e){return typeof e}:function(e){return e&&typeof Symbol=="function"&&e.constructor===Symbol&&e!==Symbol.prototype?"symbol":typeof e},F(t)}function G(t,e,n){return(e=at(e))in t?Object.defineProperty(t,e,{value:n,enumerable:!0,configurable:!0,writable:!0}):t[e]=n,t}function at(t){var e=st(t,"string");return F(e)=="symbol"?e:e+""}function st(t,e){if(F(t)!="object"||!t)return t;var n=t[Symbol.toPrimitive];if(n!==void 0){var r=n.call(t,e);if(F(r)!="object")return r;throw new TypeError("@@toPrimitive must return a primitive value.")}return(e==="string"?String:Number)(t)}var se={name:"Drawer",extends:ot,inheritAttrs:!1,emits:["update:visible","show","after-show","hide","after-hide","before-hide"],data:function(){return{containerVisible:this.visible}},container:null,mask:null,content:null,headerContainer:null,footerContainer:null,closeButton:null,outsideClickListener:null,documentKeydownListener:null,watch:{dismissable:function(e){e&&!this.modal?this.bindOutsideClickListener():this.unbindOutsideClickListener()}},updated:function(){this.visible&&(this.containerVisible=this.visible)},beforeUnmount:function(){this.disableDocumentSettings(),this.mask&&this.autoZIndex&&C.clear(this.mask),this.container=null,this.mask=null},methods:{hide:function(){this.$emit("update:visible",!1)},onEnter:function(){this.$emit("show"),this.focus(),this.bindDocumentKeyDownListener(),this.autoZIndex&&C.set("modal",this.mask,this.baseZIndex||this.$primevue.config.zIndex.modal)},onAfterEnter:function(){this.enableDocumentSettings(),this.$emit("after-show")},onBeforeLeave:function(){this.modal&&!this.isUnstyled&&he(this.mask,"p-overlay-mask-leave-active"),this.$emit("before-hide")},onLeave:function(){this.$emit("hide")},onAfterLeave:function(){this.autoZIndex&&C.clear(this.mask),this.unbindDocumentKeyDownListener(),this.containerVisible=!1,this.disableDocumentSettings(),this.$emit("after-hide")},onMaskClick:function(e){this.dismissable&&this.modal&&this.mask===e.target&&this.hide()},focus:function(){var e=function(a){return a&&a.querySelector("[autofocus]")},n=this.$slots.header&&e(this.headerContainer);n||(n=this.$slots.default&&e(this.container),n||(n=this.$slots.footer&&e(this.footerContainer),n||(n=this.closeButton))),n&&w(n)},enableDocumentSettings:function(){this.dismissable&&!this.modal&&this.bindOutsideClickListener(),this.blockScroll&&Me()},disableDocumentSettings:function(){this.unbindOutsideClickListener(),this.blockScroll&&Ke()},onKeydown:function(e){e.code==="Escape"&&this.closeOnEscape&&this.hide()},containerRef:function(e){this.container=e},maskRef:function(e){this.mask=e},contentRef:function(e){this.content=e},headerContainerRef:function(e){this.headerContainer=e},footerContainerRef:function(e){this.footerContainer=e},closeButtonRef:function(e){this.closeButton=e?e.$el:void 0},bindDocumentKeyDownListener:function(){this.documentKeydownListener||(this.documentKeydownListener=this.onKeydown,document.addEventListener("keydown",this.documentKeydownListener))},unbindDocumentKeyDownListener:function(){this.documentKeydownListener&&(document.removeEventListener("keydown",this.documentKeydownListener),this.documentKeydownListener=null)},bindOutsideClickListener:function(){var e=this;this.outsideClickListener||(this.outsideClickListener=function(n){e.isOutsideClicked(n)&&e.hide()},document.addEventListener("click",this.outsideClickListener,!0))},unbindOutsideClickListener:function(){this.outsideClickListener&&(document.removeEventListener("click",this.outsideClickListener,!0),this.outsideClickListener=null)},isOutsideClicked:function(e){return this.container&&!this.container.contains(e.target)}},computed:{fullScreen:function(){return this.position==="full"},closeAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.close:void 0},dataP:function(){return _(G(G(G({"full-screen":this.position==="full"},this.position,this.position),"open",this.containerVisible),"modal",this.modal))}},directives:{focustrap:Se},components:{Button:A,Portal:ie,TimesIcon:ze}},ut=["data-p"],lt=["role","aria-modal","data-p"];function dt(t,e,n,r,a,i){var u=S("Button"),c=S("Portal"),o=B("focustrap");return s(),k(c,null,{default:P(function(){return[a.containerVisible?(s(),m("div",d({key:0,ref:i.maskRef,onMousedown:e[0]||(e[0]=function(){return i.onMaskClick&&i.onMaskClick.apply(i,arguments)}),class:t.cx("mask"),style:t.sx("mask",!0,{position:t.position,modal:t.modal}),"data-p":i.dataP},t.ptm("mask")),[v(ne,d({name:"p-drawer",onEnter:i.onEnter,onAfterEnter:i.onAfterEnter,onBeforeLeave:i.onBeforeLeave,onLeave:i.onLeave,onAfterLeave:i.onAfterLeave,appear:""},t.ptm("transition")),{default:P(function(){return[t.visible?V((s(),m("div",d({key:0,ref:i.containerRef,class:t.cx("root"),style:t.sx("root"),role:t.modal?"dialog":"complementary","aria-modal":t.modal?!0:void 0,"data-p":i.dataP},t.ptmi("root")),[t.$slots.container?y(t.$slots,"container",{key:0,closeCallback:i.hide}):(s(),m(L,{key:1},[h("div",d({ref:i.headerContainerRef,class:t.cx("header")},t.ptm("header")),[y(t.$slots,"header",{class:O(t.cx("title"))},function(){return[t.header?(s(),m("div",d({key:0,class:t.cx("title")},t.ptm("title")),R(t.header),17)):p("",!0)]}),t.showCloseIcon?y(t.$slots,"closebutton",{key:0,closeCallback:i.hide},function(){return[v(u,d({ref:i.closeButtonRef,type:"button",class:t.cx("pcCloseButton"),"aria-label":i.closeAriaLabel,unstyled:t.unstyled,onClick:i.hide},t.closeButtonProps,{pt:t.ptm("pcCloseButton"),"data-pc-group-section":"iconcontainer"}),{icon:P(function(l){return[y(t.$slots,"closeicon",{},function(){return[(s(),k(K(t.closeIcon?"span":"TimesIcon"),d({class:[t.closeIcon,l.class]},t.ptm("pcCloseButton").icon),null,16,["class"]))]})]}),_:3},16,["class","aria-label","unstyled","onClick","pt"])]}):p("",!0)],16),h("div",d({ref:i.contentRef,class:t.cx("content")},t.ptm("content")),[y(t.$slots,"default")],16),t.$slots.footer?(s(),m("div",d({key:0,ref:i.footerContainerRef,class:t.cx("footer")},t.ptm("footer")),[y(t.$slots,"footer")],16)):p("",!0)],64))],16,lt)),[[o]]):p("",!0)]}),_:3},16,["onEnter","onAfterEnter","onBeforeLeave","onLeave","onAfterLeave"])],16,ut)):p("",!0)]}),_:3})}se.render=dt;var mt={name:"Sidebar",extends:se,mounted:function(){console.warn("Deprecated since v4. Use Drawer component instead.")}},X={name:"InputSwitch",extends:Fe,mounted:function(){console.warn("Deprecated since v4. Use ToggleSwitch component instead.")}},ct=`
    .p-menu {
        background: dt('menu.background');
        color: dt('menu.color');
        border: 1px solid dt('menu.border.color');
        border-radius: dt('menu.border.radius');
        min-width: 12.5rem;
    }

    .p-menu-list {
        margin: 0;
        padding: dt('menu.list.padding');
        outline: 0 none;
        list-style: none;
        display: flex;
        flex-direction: column;
        gap: dt('menu.list.gap');
    }

    .p-menu-item-content {
        transition:
            background dt('menu.transition.duration'),
            color dt('menu.transition.duration');
        border-radius: dt('menu.item.border.radius');
        color: dt('menu.item.color');
        overflow: hidden;
    }

    .p-menu-item-link {
        cursor: pointer;
        display: flex;
        align-items: center;
        text-decoration: none;
        overflow: hidden;
        position: relative;
        color: inherit;
        padding: dt('menu.item.padding');
        gap: dt('menu.item.gap');
        user-select: none;
        outline: 0 none;
    }

    .p-menu-item-label {
        line-height: 1;
    }

    .p-menu-item-icon {
        color: dt('menu.item.icon.color');
    }

    .p-menu-item.p-focus .p-menu-item-content {
        color: dt('menu.item.focus.color');
        background: dt('menu.item.focus.background');
    }

    .p-menu-item.p-focus .p-menu-item-icon {
        color: dt('menu.item.icon.focus.color');
    }

    .p-menu-item:not(.p-disabled) .p-menu-item-content:hover {
        color: dt('menu.item.focus.color');
        background: dt('menu.item.focus.background');
    }

    .p-menu-item:not(.p-disabled) .p-menu-item-content:hover .p-menu-item-icon {
        color: dt('menu.item.icon.focus.color');
    }

    .p-menu-overlay {
        box-shadow: dt('menu.shadow');
    }

    .p-menu-submenu-label {
        background: dt('menu.submenu.label.background');
        padding: dt('menu.submenu.label.padding');
        color: dt('menu.submenu.label.color');
        font-weight: dt('menu.submenu.label.font.weight');
    }

    .p-menu-separator {
        border-block-start: 1px solid dt('menu.separator.border.color');
    }
`,ft={root:function(e){var n=e.props;return["p-menu p-component",{"p-menu-overlay":n.popup}]},start:"p-menu-start",list:"p-menu-list",submenuLabel:"p-menu-submenu-label",separator:"p-menu-separator",end:"p-menu-end",item:function(e){var n=e.instance;return["p-menu-item",{"p-focus":n.id===n.focusedOptionId,"p-disabled":n.disabled()}]},itemContent:"p-menu-item-content",itemLink:"p-menu-item-link",itemIcon:"p-menu-item-icon",itemLabel:"p-menu-item-label"},bt=q.extend({name:"menu",style:ct,classes:ft}),pt={name:"BaseMenu",extends:T,props:{popup:{type:Boolean,default:!1},model:{type:Array,default:null},appendTo:{type:[String,Object],default:"body"},autoZIndex:{type:Boolean,default:!0},baseZIndex:{type:Number,default:0},tabindex:{type:Number,default:0},ariaLabel:{type:String,default:null},ariaLabelledby:{type:String,default:null}},style:bt,provide:function(){return{$pcMenu:this,$parentInstance:this}}},ue={name:"Menuitem",hostName:"Menu",extends:T,inheritAttrs:!1,emits:["item-click","item-mousemove"],props:{item:null,templates:null,id:null,focusedOptionId:null,index:null},methods:{getItemProp:function(e,n){return e&&e.item?W(e.item[n]):void 0},getPTOptions:function(e){return this.ptm(e,{context:{item:this.item,index:this.index,focused:this.isItemFocused(),disabled:this.disabled()}})},isItemFocused:function(){return this.focusedOptionId===this.id},onItemClick:function(e){var n=this.getItemProp(this.item,"command");n&&n({originalEvent:e,item:this.item.item}),this.$emit("item-click",{originalEvent:e,item:this.item,id:this.id})},onItemMouseMove:function(e){this.$emit("item-mousemove",{originalEvent:e,item:this.item,id:this.id})},visible:function(){return typeof this.item.visible=="function"?this.item.visible():this.item.visible!==!1},disabled:function(){return typeof this.item.disabled=="function"?this.item.disabled():this.item.disabled},label:function(){return typeof this.item.label=="function"?this.item.label():this.item.label},getMenuItemProps:function(e){return{action:d({class:this.cx("itemLink"),tabindex:"-1"},this.getPTOptions("itemLink")),icon:d({class:[this.cx("itemIcon"),e.icon]},this.getPTOptions("itemIcon")),label:d({class:this.cx("itemLabel")},this.getPTOptions("itemLabel"))}}},computed:{dataP:function(){return _({focus:this.isItemFocused(),disabled:this.disabled()})}},directives:{ripple:re}},ht=["id","aria-label","aria-disabled","data-p-focused","data-p-disabled","data-p"],vt=["data-p"],It=["href","target"],gt=["data-p"],yt=["data-p"];function kt(t,e,n,r,a,i){var u=B("ripple");return i.visible()?(s(),m("li",d({key:0,id:n.id,class:[t.cx("item"),n.item.class],role:"menuitem",style:n.item.style,"aria-label":i.label(),"aria-disabled":i.disabled(),"data-p-focused":i.isItemFocused(),"data-p-disabled":i.disabled()||!1,"data-p":i.dataP},i.getPTOptions("item")),[h("div",d({class:t.cx("itemContent"),onClick:e[0]||(e[0]=function(c){return i.onItemClick(c)}),onMousemove:e[1]||(e[1]=function(c){return i.onItemMouseMove(c)}),"data-p":i.dataP},i.getPTOptions("itemContent")),[n.templates.item?n.templates.item?(s(),k(K(n.templates.item),{key:1,item:n.item,label:i.label(),props:i.getMenuItemProps(n.item)},null,8,["item","label","props"])):p("",!0):V((s(),m("a",d({key:0,href:n.item.url,class:t.cx("itemLink"),target:n.item.target,tabindex:"-1"},i.getPTOptions("itemLink")),[n.templates.itemicon?(s(),k(K(n.templates.itemicon),{key:0,item:n.item,class:O(t.cx("itemIcon"))},null,8,["item","class"])):n.item.icon?(s(),m("span",d({key:1,class:[t.cx("itemIcon"),n.item.icon],"data-p":i.dataP},i.getPTOptions("itemIcon")),null,16,gt)):p("",!0),h("span",d({class:t.cx("itemLabel"),"data-p":i.dataP},i.getPTOptions("itemLabel")),R(i.label()),17,yt)],16,It)),[[u]])],16,vt)],16,ht)):p("",!0)}ue.render=kt;function $(t){return Pt(t)||xt(t)||Lt(t)||wt()}function wt(){throw new TypeError(`Invalid attempt to spread non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`)}function Lt(t,e){if(t){if(typeof t=="string")return Z(t,e);var n={}.toString.call(t).slice(8,-1);return n==="Object"&&t.constructor&&(n=t.constructor.name),n==="Map"||n==="Set"?Array.from(t):n==="Arguments"||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n)?Z(t,e):void 0}}function xt(t){if(typeof Symbol<"u"&&t[Symbol.iterator]!=null||t["@@iterator"]!=null)return Array.from(t)}function Pt(t){if(Array.isArray(t))return Z(t)}function Z(t,e){(e==null||e>t.length)&&(e=t.length);for(var n=0,r=Array(e);n<e;n++)r[n]=t[n];return r}var le={name:"Menu",extends:pt,inheritAttrs:!1,emits:["show","hide","focus","blur"],data:function(){return{overlayVisible:!1,focused:!1,focusedOptionIndex:-1,selectedOptionIndex:-1}},target:null,outsideClickListener:null,scrollHandler:null,resizeListener:null,container:null,list:null,mounted:function(){this.popup||(this.bindResizeListener(),this.bindOutsideClickListener())},beforeUnmount:function(){this.unbindResizeListener(),this.unbindOutsideClickListener(),this.scrollHandler&&(this.scrollHandler.destroy(),this.scrollHandler=null),this.target=null,this.container&&this.autoZIndex&&C.clear(this.container),this.container=null},methods:{itemClick:function(e){var n=e.item;this.disabled(n)||(n.command&&n.command(e),this.overlayVisible&&this.hide(),!this.popup&&this.focusedOptionIndex!==e.id&&(this.focusedOptionIndex=e.id))},itemMouseMove:function(e){this.focused&&(this.focusedOptionIndex=e.id)},onListFocus:function(e){this.focused=!0,!this.popup&&this.changeFocusedOptionIndex(0),this.$emit("focus",e)},onListBlur:function(e){this.focused=!1,this.focusedOptionIndex=-1,this.$emit("blur",e)},onListKeyDown:function(e){switch(e.code){case"ArrowDown":this.onArrowDownKey(e);break;case"ArrowUp":this.onArrowUpKey(e);break;case"Home":this.onHomeKey(e);break;case"End":this.onEndKey(e);break;case"Enter":case"NumpadEnter":this.onEnterKey(e);break;case"Space":this.onSpaceKey(e);break;case"Escape":this.popup&&(w(this.target),this.hide());case"Tab":this.overlayVisible&&this.hide();break}},onArrowDownKey:function(e){var n=this.findNextOptionIndex(this.focusedOptionIndex);this.changeFocusedOptionIndex(n),e.preventDefault()},onArrowUpKey:function(e){if(e.altKey&&this.popup)w(this.target),this.hide(),e.preventDefault();else{var n=this.findPrevOptionIndex(this.focusedOptionIndex);this.changeFocusedOptionIndex(n),e.preventDefault()}},onHomeKey:function(e){this.changeFocusedOptionIndex(0),e.preventDefault()},onEndKey:function(e){this.changeFocusedOptionIndex(N(this.container,'li[data-pc-section="item"][data-p-disabled="false"]').length-1),e.preventDefault()},onEnterKey:function(e){var n=E(this.list,'li[id="'.concat("".concat(this.focusedOptionIndex),'"]')),r=n&&E(n,'a[data-pc-section="itemlink"]');this.popup&&w(this.target),r?r.click():n&&n.click(),e.preventDefault()},onSpaceKey:function(e){this.onEnterKey(e)},findNextOptionIndex:function(e){var n=N(this.container,'li[data-pc-section="item"][data-p-disabled="false"]'),r=$(n).findIndex(function(a){return a.id===e});return r>-1?r+1:0},findPrevOptionIndex:function(e){var n=N(this.container,'li[data-pc-section="item"][data-p-disabled="false"]'),r=$(n).findIndex(function(a){return a.id===e});return r>-1?r-1:0},changeFocusedOptionIndex:function(e){var n=N(this.container,'li[data-pc-section="item"][data-p-disabled="false"]'),r=e>=n.length?n.length-1:e<0?0:e;r>-1&&(this.focusedOptionIndex=n[r].getAttribute("id"))},toggle:function(e,n){this.overlayVisible?this.hide():this.show(e,n)},show:function(e,n){this.overlayVisible=!0,this.target=n??e.currentTarget},hide:function(){this.overlayVisible=!1,this.target=null},onEnter:function(e){Ie(e,{position:"absolute",top:"0"}),this.alignOverlay(),this.bindOutsideClickListener(),this.bindResizeListener(),this.bindScrollListener(),this.autoZIndex&&C.set("menu",e,this.baseZIndex+this.$primevue.config.zIndex.menu),this.popup&&w(this.list),this.$emit("show")},onLeave:function(){this.unbindOutsideClickListener(),this.unbindResizeListener(),this.unbindScrollListener(),this.$emit("hide")},onAfterLeave:function(e){this.autoZIndex&&C.clear(e)},alignOverlay:function(){ve(this.container,this.target);var e=H(this.target);e>H(this.container)&&(this.container.style.minWidth=H(this.target)+"px")},bindOutsideClickListener:function(){var e=this;this.outsideClickListener||(this.outsideClickListener=function(n){var r=e.container&&!e.container.contains(n.target),a=!(e.target&&(e.target===n.target||e.target.contains(n.target)));e.overlayVisible&&r&&a?e.hide():!e.popup&&r&&a&&(e.focusedOptionIndex=-1)},document.addEventListener("click",this.outsideClickListener,!0))},unbindOutsideClickListener:function(){this.outsideClickListener&&(document.removeEventListener("click",this.outsideClickListener,!0),this.outsideClickListener=null)},bindScrollListener:function(){var e=this;this.scrollHandler||(this.scrollHandler=new Ve(this.target,function(){e.overlayVisible&&e.hide()})),this.scrollHandler.bindScrollListener()},unbindScrollListener:function(){this.scrollHandler&&this.scrollHandler.unbindScrollListener()},bindResizeListener:function(){var e=this;this.resizeListener||(this.resizeListener=function(){e.overlayVisible&&!te()&&e.hide()},window.addEventListener("resize",this.resizeListener))},unbindResizeListener:function(){this.resizeListener&&(window.removeEventListener("resize",this.resizeListener),this.resizeListener=null)},visible:function(e){return typeof e.visible=="function"?e.visible():e.visible!==!1},disabled:function(e){return typeof e.disabled=="function"?e.disabled():e.disabled},label:function(e){return typeof e.label=="function"?e.label():e.label},onOverlayClick:function(e){Be.emit("overlay-click",{originalEvent:e,target:this.target})},containerRef:function(e){this.container=e},listRef:function(e){this.list=e}},computed:{focusedOptionId:function(){return this.focusedOptionIndex!==-1?this.focusedOptionIndex:null},dataP:function(){return _({popup:this.popup})}},components:{PVMenuitem:ue,Portal:ie}},Ct=["id","data-p"],Ot=["id","tabindex","aria-activedescendant","aria-label","aria-labelledby"],St=["id"];function Kt(t,e,n,r,a,i){var u=S("PVMenuitem"),c=S("Portal");return s(),k(c,{appendTo:t.appendTo,disabled:!t.popup},{default:P(function(){return[v(ne,d({name:"p-anchored-overlay",onEnter:i.onEnter,onLeave:i.onLeave,onAfterLeave:i.onAfterLeave},t.ptm("transition")),{default:P(function(){return[!t.popup||a.overlayVisible?(s(),m("div",d({key:0,ref:i.containerRef,id:t.$id,class:t.cx("root"),onClick:e[3]||(e[3]=function(){return i.onOverlayClick&&i.onOverlayClick.apply(i,arguments)}),"data-p":i.dataP},t.ptmi("root")),[t.$slots.start?(s(),m("div",d({key:0,class:t.cx("start")},t.ptm("start")),[y(t.$slots,"start")],16)):p("",!0),h("ul",d({ref:i.listRef,id:t.$id+"_list",class:t.cx("list"),role:"menu",tabindex:t.tabindex,"aria-activedescendant":a.focused?i.focusedOptionId:void 0,"aria-label":t.ariaLabel,"aria-labelledby":t.ariaLabelledby,onFocus:e[0]||(e[0]=function(){return i.onListFocus&&i.onListFocus.apply(i,arguments)}),onBlur:e[1]||(e[1]=function(){return i.onListBlur&&i.onListBlur.apply(i,arguments)}),onKeydown:e[2]||(e[2]=function(){return i.onListKeyDown&&i.onListKeyDown.apply(i,arguments)})},t.ptm("list")),[(s(!0),m(L,null,U(t.model,function(o,l){return s(),m(L,{key:i.label(o)+l.toString()},[o.items&&i.visible(o)&&!o.separator?(s(),m(L,{key:0},[o.items?(s(),m("li",d({key:0,id:t.$id+"_"+l,class:[t.cx("submenuLabel"),o.class],role:"none"},{ref_for:!0},t.ptm("submenuLabel")),[y(t.$slots,t.$slots.submenulabel?"submenulabel":"submenuheader",{item:o},function(){return[ge(R(i.label(o)),1)]})],16,St)):p("",!0),(s(!0),m(L,null,U(o.items,function(f,b){return s(),m(L,{key:f.label+l+"_"+b},[i.visible(f)&&!f.separator?(s(),k(u,{key:0,id:t.$id+"_"+l+"_"+b,item:f,templates:t.$slots,focusedOptionId:i.focusedOptionId,unstyled:t.unstyled,onItemClick:i.itemClick,onItemMousemove:i.itemMouseMove,pt:t.pt},null,8,["id","item","templates","focusedOptionId","unstyled","onItemClick","onItemMousemove","pt"])):i.visible(f)&&f.separator?(s(),m("li",d({key:"separator"+l+b,class:[t.cx("separator"),o.class],style:f.style,role:"separator"},{ref_for:!0},t.ptm("separator")),null,16)):p("",!0)],64)}),128))],64)):i.visible(o)&&o.separator?(s(),m("li",d({key:"separator"+l.toString(),class:[t.cx("separator"),o.class],style:o.style,role:"separator"},{ref_for:!0},t.ptm("separator")),null,16)):(s(),k(u,{key:i.label(o)+l.toString(),id:t.$id+"_"+l,item:o,index:l,templates:t.$slots,focusedOptionId:i.focusedOptionId,unstyled:t.unstyled,onItemClick:i.itemClick,onItemMousemove:i.itemMouseMove,pt:t.pt},null,8,["id","item","index","templates","focusedOptionId","unstyled","onItemClick","onItemMousemove","pt"]))],64)}),128))],16,Ot),t.$slots.end?(s(),m("div",d({key:1,class:t.cx("end")},t.ptm("end")),[y(t.$slots,"end")],16)):p("",!0)],16,Ct)):p("",!0)]}),_:3},16,["onEnter","onLeave","onAfterLeave"])]}),_:3},8,["appendTo","disabled"])}le.render=Kt;const Mt={class:"language-selector"},At={__name:"LanguageSelector",setup(t){const{locale:e}=ye(),n=D(),r=ke(()=>e.value==="de"?"pi pi-flag":"pi pi-globe"),a=D([{label:"Deutsch",icon:"pi pi-flag",command:()=>u("de")},{label:"English",icon:"pi pi-globe",command:()=>u("en")}]),i=c=>{n.value.toggle(c)},u=c=>{e.value=c,we(c)};return(c,o)=>{const l=B("tooltip");return s(),m("div",Mt,[V(v(I(A),{icon:r.value,onClick:i,rounded:"",severity:"secondary",class:"lang-btn"},null,8,["icon"]),[[l,c.$t("common.language"),void 0,{bottom:!0}]]),v(I(le),{ref_key:"menu",ref:n,model:a.value,popup:!0},null,8,["model"])])}}},ee=Re(At,[["__scopeId","data-v-387ddd44"]]),Et={class:"min-h-screen flex flex-col"},Dt={class:"flex items-center gap-4"},zt={class:"flex items-center gap-2"},Ft={class:"hidden sm:flex items-center gap-2 px-3"},Bt={class:"flex flex-col gap-2 h-full"},Vt={class:"mt-auto border-t border-gray-700 pt-4 flex flex-col gap-4"},Rt={class:"flex items-center justify-between px-3"},Tt={class:"flex items-center gap-2"},jt={class:"flex-grow bg-gray-900 text-white"},Qt={__name:"Layout",setup(t){const e=Oe(),n=Le(),r=xe(),a=D(!1),i=D(!1),u=f=>{e.push(f),a.value=!1},c=D([{label:"Dashboard",icon:"pi pi-home",command:()=>u("/")},{label:"Control",icon:"pi pi-sliders-h",command:()=>u("/control")},{label:"Devices",icon:"pi pi-desktop",command:()=>u("/devices")},{label:"Logs",icon:"pi pi-list",command:()=>u("/logs")},{label:"System",icon:"pi pi-info-circle",command:()=>u("/system")},{label:"Settings",icon:"pi pi-cog",command:()=>u("/config")}]),o=async()=>{await n.logout(),e.push("/login")},l=()=>{i.value=window.innerWidth<768};return Pe(()=>{l(),window.addEventListener("resize",l)}),Ce(()=>{window.removeEventListener("resize",l)}),(f,b)=>{const x=S("router-view"),de=B("ripple");return s(),m("div",Et,[v(I(ae),{model:i.value?[]:c.value,class:"rounded-none border-0 border-b border-gray-700 bg-gray-800"},{start:P(()=>[h("div",Dt,[i.value?(s(),k(I(A),{key:0,icon:"pi pi-bars",text:"",rounded:"",onClick:b[0]||(b[0]=g=>a.value=!0),class:"text-white"})):p("",!0),b[4]||(b[4]=h("span",{class:"text-xl font-bold px-4 text-white"},"ModBridge",-1))])]),item:P(({item:g,props:me})=>[V((s(),m("a",d({class:"flex items-center gap-2 px-3 py-2 hover:bg-gray-700 rounded cursor-pointer text-gray-200"},me.action),[h("i",{class:O(g.icon)},null,2),h("span",null,R(g.label),1)],16)),[[de]])]),end:P(()=>[h("div",zt,[v(ee,{class:"hidden sm:flex"}),h("div",Ft,[h("i",{class:O(I(r).darkMode?"pi pi-moon":"pi pi-sun")},null,2),v(I(X),{modelValue:I(r).darkMode,"onUpdate:modelValue":b[1]||(b[1]=g=>I(r).toggleDarkMode(g))},null,8,["modelValue"])]),v(I(A),{label:"Logout",icon:"pi pi-power-off",severity:"danger",text:"",onClick:o,class:"hidden sm:flex"})])]),_:1},8,["model"]),v(I(mt),{visible:a.value,"onUpdate:visible":b[3]||(b[3]=g=>a.value=g),baseZIndex:1e4},{default:P(()=>[h("div",Bt,[(s(!0),m(L,null,U(c.value,g=>(s(),m("div",{key:g.label},[v(I(A),{onClick:g.command,label:g.label,icon:g.icon,text:"",class:"w-full text-left",size:"large"},null,8,["onClick","label","icon"])]))),128)),h("div",Vt,[h("div",Rt,[b[5]||(b[5]=h("span",{class:"text-gray-300 font-medium"},"Theme",-1)),h("div",Tt,[h("i",{class:O(I(r).darkMode?"pi pi-moon text-gray-300":"pi pi-sun text-gray-300")},null,2),v(I(X),{modelValue:I(r).darkMode,"onUpdate:modelValue":b[2]||(b[2]=g=>I(r).toggleDarkMode(g))},null,8,["modelValue"])])]),v(ee,{class:"w-full px-3"}),v(I(A),{onClick:o,label:"Logout",icon:"pi pi-power-off",severity:"danger",text:"",class:"w-full text-left",size:"large"})])])]),_:1},8,["visible"]),h("main",jt,[v(x)])])}}};export{Qt as default};

# LLM Verification Report

Generated on: 2025-12-23 19:25:02

## Summary

- Total Models: 9
- Available Models: 1
- Failed Models: 8
- Average Overall Score: 51.91

### Top Performers by Overall Score

1. **deepseek-chat**: 48.00

## Model: gpt2 (FAILED)

**Error**: failed to check model existence: API request failed with status 410: <!doctype html>
<html class="">
	<head>
		<meta charset="utf-8" />

		<meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no" />

		<meta name="description" content="We’re on a journey to advance and democratize artificial intelligence through open source and open science." />

		<meta property="fb:app_id" content="1321688464574422" />

		<meta name="twitter:card" content="summary_large_image" />

		<meta name="twitter:site" content="@huggingface" />

		<meta name="twitter:image" content="https://huggingface.co/front/thumbnails/models.png" />

		<meta property="og:title" content="410 – Hugging Face" />

		<meta property="og:type" content="website" />

		<meta property="og:url" content="https://huggingface.co/models" />

		<meta property="og:image" content="https://huggingface.co/front/thumbnails/models.png" />

		<link rel="stylesheet" href="/front/build/kube-3c60e60/style.css" />

		<link rel="preconnect" href="https://fonts.gstatic.com" />

		<link
			href="https://fonts.googleapis.com/css2?family=Source+Sans+Pro:ital,wght@0,200;0,300;0,400;0,600;0,700;1,200;1,300;1,400;1,600;1,700&display=swap"
			rel="stylesheet"
		/>

		<link
			href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;600;700&display=swap"
			rel="stylesheet"
		/>

		<link
			rel="preload"
			href="https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.12.0/katex.min.css"
			as="style"
			onload="this.onload=null;this.rel='stylesheet'"
		/>

		<noscript>
			<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.12.0/katex.min.css" />
		</noscript>
		<script>const guestTheme = document.cookie.match(/theme=(\w+)/)?.[1]; document.documentElement.classList.toggle('dark', guestTheme === 'dark' || ( (!guestTheme || guestTheme === 'system') && window.matchMedia('(prefers-color-scheme: dark)').matches));</script>
<link rel="canonical" href="https://huggingface.co/models">  
		<title>410 – Hugging Face</title>

		<script defer src="/js/script.js"></script>

		<script>
			(window.plausible =
				window.plausible ||
				function () {
					(plausible.q = plausible.q || []).push(arguments);
				}),
				(plausible.init =
					plausible.init ||
					function (i) {
						plausible.o = i || {};
					});
			plausible.init({
				customProperties: {
					loggedIn: "false",
				},
				endpoint: "/api/event",
			});
		</script>

		<script>
			window.hubConfig = {"features":{"signupDisabled":false},"sshGitUrl":"git@hf.co","moonHttpUrl":"https:\/\/huggingface.co","captchaApiKey":"bd5f2066-93dc-4bdd-a64b-a24646ca3859","datasetViewerPublicUrl":"https:\/\/datasets-server.huggingface.co","stripePublicKey":"pk_live_x2tdjFXBCvXo2FFmMybezpeM00J6gPCAAc","environment":"production","userAgent":"HuggingFace (production)","spacesIframeDomain":"hf.space","spacesApiUrl":"https:\/\/api.hf.space","docSearchKey":"ece5e02e57300e17d152c08056145326e90c4bff3dd07d7d1ae40cf1c8d39cb6","logoDev":{"apiUrl":"https:\/\/img.logo.dev\/","apiKey":"pk_UHS2HZOeRnaSOdDp7jbd5w"}};
			window.requestId = "";
		</script>
		<script type="text/javascript" src="https://de5282c3ca0c.edge.sdk.awswaf.com/de5282c3ca0c/526cf06acb0d/challenge.js" defer></script> 
	</head>
	<body class="flex flex-col min-h-dvh bg-white dark:bg-gray-950 text-black ErrorPage">
		<div class="flex min-h-dvh flex-col"><div class="SVELTE_HYDRATER contents" data-target="DeviceProvider" data-props="{}"></div>
	<div class="SVELTE_HYDRATER contents" data-target="SystemThemeMonitor" data-props="{&quot;isLoggedIn&quot;:false}"></div>

	<div class="SVELTE_HYDRATER contents" data-target="MainHeader" data-props="{&quot;classNames&quot;:&quot;&quot;,&quot;isWide&quot;:false,&quot;isZh&quot;:false,&quot;isPro&quot;:false}"><header class="border-b border-gray-100 "><div class="w-full px-4 container flex h-16 items-center"><div class="flex flex-1 items-center"><a class="mr-5 flex flex-none items-center lg:mr-6" href="/"><img alt="Hugging Face's logo" class="w-7 md:mr-2" src="/front/assets/huggingface_logo-noborder.svg">
				<span class="hidden whitespace-nowrap text-lg font-bold md:block">Hugging Face</span></a>
			<div class="relative flex-1 lg:max-w-sm mr-2 sm:mr-4 md:mr-3 xl:mr-6"><input autocomplete="off" class="w-full dark:bg-gray-950 pl-8 form-input-alt h-9 pr-3 focus:shadow-xl " name="" placeholder="Search models, datasets, users..."   spellcheck="false" type="text" value="">
	<svg class="absolute left-2.5 text-gray-400 top-1/2 transform -translate-y-1/2" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 32 32"><path d="M30 28.59L22.45 21A11 11 0 1 0 21 22.45L28.59 30zM5 14a9 9 0 1 1 9 9a9 9 0 0 1-9-9z" fill="currentColor"></path></svg>
	</div>
			<div class="flex flex-none items-center justify-center p-0.5 place-self-stretch lg:hidden"><button class="relative z-40 flex h-6 w-8 items-center justify-center" type="button"><svg width="1em" height="1em" viewBox="0 0 10 10" class="text-xl" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" preserveAspectRatio="xMidYMid meet" fill="currentColor"><path fill-rule="evenodd" clip-rule="evenodd" d="M1.65039 2.9999C1.65039 2.8066 1.80709 2.6499 2.00039 2.6499H8.00039C8.19369 2.6499 8.35039 2.8066 8.35039 2.9999C8.35039 3.1932 8.19369 3.3499 8.00039 3.3499H2.00039C1.80709 3.3499 1.65039 3.1932 1.65039 2.9999ZM1.65039 4.9999C1.65039 4.8066 1.80709 4.6499 2.00039 4.6499H8.00039C8.19369 4.6499 8.35039 4.8066 8.35039 4.9999C8.35039 5.1932 8.19369 5.3499 8.00039 5.3499H2.00039C1.80709 5.3499 1.65039 5.1932 1.65039 4.9999ZM2.00039 6.6499C1.80709 6.6499 1.65039 6.8066 1.65039 6.9999C1.65039 7.1932 1.80709 7.3499 2.00039 7.3499H8.00039C8.19369 7.3499 8.35039 7.1932 8.35039 6.9999C8.35039 6.8066 8.19369 6.6499 8.00039 6.6499H2.00039Z"></path></svg>
		</button>

	</div></div>
		<nav aria-label="Main" class="ml-auto hidden lg:block"><ul class="flex items-center gap-x-1 2xl:gap-x-2"><li class="hover:text-indigo-700"><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/models"><svg class="mr-1.5 text-gray-400 group-hover:text-indigo-500" style="" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 24 24"><path class="uim-quaternary" d="M20.23 7.24L12 12L3.77 7.24a1.98 1.98 0 0 1 .7-.71L11 2.76c.62-.35 1.38-.35 2 0l6.53 3.77c.29.173.531.418.7.71z" opacity=".25" fill="currentColor"></path><path class="uim-tertiary" d="M12 12v9.5a2.09 2.09 0 0 1-.91-.21L4.5 17.48a2.003 2.003 0 0 1-1-1.73v-7.5a2.06 2.06 0 0 1 .27-1.01L12 12z" opacity=".5" fill="currentColor"></path><path class="uim-primary" d="M20.5 8.25v7.5a2.003 2.003 0 0 1-1 1.73l-6.62 3.82c-.275.13-.576.198-.88.2V12l8.23-4.76c.175.308.268.656.27 1.01z" fill="currentColor"></path></svg>
						Models</a>
				</li><li class="hover:text-red-700"><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/datasets"><svg class="mr-1.5 text-gray-400 group-hover:text-red-500" style="" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 25 25"><ellipse cx="12.5" cy="5" fill="currentColor" fill-opacity="0.25" rx="7.5" ry="2"></ellipse><path d="M12.5 15C16.6421 15 20 14.1046 20 13V20C20 21.1046 16.6421 22 12.5 22C8.35786 22 5 21.1046 5 20V13C5 14.1046 8.35786 15 12.5 15Z" fill="currentColor" opacity="0.5"></path><path d="M12.5 7C16.6421 7 20 6.10457 20 5V11.5C20 12.6046 16.6421 13.5 12.5 13.5C8.35786 13.5 5 12.6046 5 11.5V5C5 6.10457 8.35786 7 12.5 7Z" fill="currentColor" opacity="0.5"></path><path d="M5.23628 12C5.08204 12.1598 5 12.8273 5 13C5 14.1046 8.35786 15 12.5 15C16.6421 15 20 14.1046 20 13C20 12.8273 19.918 12.1598 19.7637 12C18.9311 12.8626 15.9947 13.5 12.5 13.5C9.0053 13.5 6.06886 12.8626 5.23628 12Z" fill="currentColor"></path></svg>
						Datasets</a>
				</li><li class="hover:text-blue-700"><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/spaces"><svg class="mr-1.5 text-gray-400 group-hover:text-blue-500" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" viewBox="0 0 25 25"><path opacity=".5" d="M6.016 14.674v4.31h4.31v-4.31h-4.31ZM14.674 14.674v4.31h4.31v-4.31h-4.31ZM6.016 6.016v4.31h4.31v-4.31h-4.31Z" fill="currentColor"></path><path opacity=".75" fill-rule="evenodd" clip-rule="evenodd" d="M3 4.914C3 3.857 3.857 3 4.914 3h6.514c.884 0 1.628.6 1.848 1.414a5.171 5.171 0 0 1 7.31 7.31c.815.22 1.414.964 1.414 1.848v6.514A1.914 1.914 0 0 1 20.086 22H4.914A1.914 1.914 0 0 1 3 20.086V4.914Zm3.016 1.102v4.31h4.31v-4.31h-4.31Zm0 12.968v-4.31h4.31v4.31h-4.31Zm8.658 0v-4.31h4.31v4.31h-4.31Zm0-10.813a2.155 2.155 0 1 1 4.31 0 2.155 2.155 0 0 1-4.31 0Z" fill="currentColor"></path><path opacity=".25" d="M16.829 6.016a2.155 2.155 0 1 0 0 4.31 2.155 2.155 0 0 0 0-4.31Z" fill="currentColor"></path></svg>
						Spaces</a>
				</li><li class="max-xl:hidden relative"><div class="relative ">
	<button class="group flex items-center px-2 py-0.5 dark:text-gray-300 hover:text-yellow-700 dark:hover:text-gray-100 " type="button">
		<svg class="mr-1.5 mr-1.5 text-gray-400 text-yellow-500! group-hover:text-yellow-500" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 32 32"><path d="M20.6081 3C21.7684 3 22.8053 3.49196 23.5284 4.38415C23.9756 4.93678 24.4428 5.82749 24.4808 7.16133C24.9674 7.01707 25.4353 6.93643 25.8725 6.93643C26.9833 6.93643 27.9865 7.37587 28.696 8.17411C29.6075 9.19872 30.0124 10.4579 29.8361 11.7177C29.7523 12.3177 29.5581 12.8555 29.2678 13.3534C29.8798 13.8646 30.3306 14.5763 30.5485 15.4322C30.719 16.1032 30.8939 17.5006 29.9808 18.9403C30.0389 19.0342 30.0934 19.1319 30.1442 19.2318C30.6932 20.3074 30.7283 21.5229 30.2439 22.6548C29.5093 24.3704 27.6841 25.7219 24.1397 27.1727C21.9347 28.0753 19.9174 28.6523 19.8994 28.6575C16.9842 29.4379 14.3477 29.8345 12.0653 29.8345C7.87017 29.8345 4.8668 28.508 3.13831 25.8921C0.356375 21.6797 0.754104 17.8269 4.35369 14.1131C6.34591 12.058 7.67023 9.02782 7.94613 8.36275C8.50224 6.39343 9.97271 4.20438 12.4172 4.20438H12.4179C12.6236 4.20438 12.8314 4.2214 13.0364 4.25468C14.107 4.42854 15.0428 5.06476 15.7115 6.02205C16.4331 5.09583 17.134 4.359 17.7682 3.94323C18.7242 3.31737 19.6794 3 20.6081 3ZM20.6081 5.95917C20.2427 5.95917 19.7963 6.1197 19.3039 6.44225C17.7754 7.44319 14.8258 12.6772 13.7458 14.7131C13.3839 15.3952 12.7655 15.6837 12.2086 15.6837C11.1036 15.6837 10.2408 14.5497 12.1076 13.1085C14.9146 10.9402 13.9299 7.39584 12.5898 7.1776C12.5311 7.16799 12.4731 7.16355 12.4172 7.16355C11.1989 7.16355 10.6615 9.33114 10.6615 9.33114C10.6615 9.33114 9.0863 13.4148 6.38031 16.206C3.67434 18.998 3.5346 21.2388 5.50675 24.2246C6.85185 26.2606 9.42666 26.8753 12.0653 26.8753C14.8021 26.8753 17.6077 26.2139 19.1799 25.793C19.2574 25.7723 28.8193 22.984 27.6081 20.6107C27.4046 20.212 27.0693 20.0522 26.6471 20.0522C24.9416 20.0522 21.8393 22.6726 20.5057 22.6726C20.2076 22.6726 19.9976 22.5416 19.9116 22.222C19.3433 20.1173 28.552 19.2325 27.7758 16.1839C27.639 15.6445 27.2677 15.4256 26.746 15.4263C24.4923 15.4263 19.4358 19.5181 18.3759 19.5181C18.2949 19.5181 18.2368 19.4937 18.2053 19.4419C17.6743 18.557 17.9653 17.9394 21.7082 15.6009C25.4511 13.2617 28.0783 11.8545 26.5841 10.1752C26.4121 9.98141 26.1684 9.8956 25.8725 9.8956C23.6001 9.89634 18.2311 14.9403 18.2311 14.9403C18.2311 14.9403 16.7821 16.496 15.9057 16.496C15.7043 16.496 15.533 16.4139 15.4169 16.2112C14.7956 15.1296 21.1879 10.1286 21.5484 8.06535C21.7928 6.66715 21.3771 5.95917 20.6081 5.95917Z" fill="#FF9D00"></path><path d="M5.50686 24.2246C3.53472 21.2387 3.67446 18.9979 6.38043 16.206C9.08641 13.4147 10.6615 9.33111 10.6615 9.33111C10.6615 9.33111 11.2499 6.95933 12.59 7.17757C13.93 7.39581 14.9139 10.9401 12.1069 13.1084C9.29997 15.276 12.6659 16.7489 13.7459 14.713C14.8258 12.6772 17.7747 7.44316 19.304 6.44221C20.8326 5.44128 21.9089 6.00204 21.5484 8.06532C21.188 10.1286 14.795 15.1295 15.4171 16.2118C16.0391 17.2934 18.2312 14.9402 18.2312 14.9402C18.2312 14.9402 25.0907 8.49588 26.5842 10.1752C28.0776 11.8545 25.4512 13.2616 21.7082 15.6008C17.9646 17.9393 17.6744 18.557 18.2054 19.4418C18.7372 20.3266 26.9998 13.1351 27.7759 16.1838C28.5513 19.2324 19.3434 20.1173 19.9117 22.2219C20.48 24.3274 26.3979 18.2382 27.6082 20.6107C28.8193 22.9839 19.2574 25.7722 19.18 25.7929C16.0914 26.62 8.24723 28.3726 5.50686 24.2246Z" fill="#FFD21E"></path></svg>
			Community
		</button>
	
	
	</div>
				</li><li class="hover:text-yellow-700"><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/docs"><svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" role="img" class="mr-1.5 text-gray-400 group-hover:text-yellow-500" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 16 16"><path d="m2.28 3.7-.3.16a.67.67 0 0 0-.34.58v8.73l.01.04.02.07.01.04.03.06.02.04.02.03.04.06.05.05.04.04.06.04.06.04.08.04.08.02h.05l.07.02h.11l.04-.01.07-.02.03-.01.07-.03.22-.12a5.33 5.33 0 0 1 5.15.1.67.67 0 0 0 .66 0 5.33 5.33 0 0 1 5.33 0 .67.67 0 0 0 1-.58V4.36a.67.67 0 0 0-.34-.5l-.3-.17v7.78a.63.63 0 0 1-.87.59 4.9 4.9 0 0 0-4.35.35l-.65.39a.29.29 0 0 1-.15.04.29.29 0 0 1-.16-.04l-.65-.4a4.9 4.9 0 0 0-4.34-.34.63.63 0 0 1-.87-.59V3.7Z" fill="currentColor" class="dark:opacity-40"></path><path fill-rule="evenodd" clip-rule="evenodd" d="M8 3.1a5.99 5.99 0 0 0-5.3-.43.66.66 0 0 0-.42.62v8.18c0 .45.46.76.87.59a4.9 4.9 0 0 1 4.34.35l.65.39c.05.03.1.04.16.04.05 0 .1-.01.15-.04l.65-.4a4.9 4.9 0 0 1 4.35-.34.63.63 0 0 0 .86-.59V3.3a.67.67 0 0 0-.41-.62 5.99 5.99 0 0 0-5.3.43l-.3.17L8 3.1Zm.73 1.87a.43.43 0 1 0-.86 0v5.48a.43.43 0 0 0 .86 0V4.97Z" fill="currentColor" class="opacity-40 dark:opacity-100"></path><path d="M8.73 4.97a.43.43 0 1 0-.86 0v5.48a.43.43 0 1 0 .86 0V4.96Z" fill="currentColor" class="dark:opacity-40"></path></svg>
						Docs</a>
				</li><li class="hover:text-black dark:hover:text-white max-2xl:hidden"><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/enterprise"><svg class="mr-1.5 text-gray-400 group-hover:text-black dark:group-hover:text-white" xmlns="http://www.w3.org/2000/svg" fill="none" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 12 12"><path fill-rule="evenodd" clip-rule="evenodd" d="M4.9 1.35a3.16 3.16 0 0 0-2.8 2.07L.37 8.58C0 9.71.7 10.65 1.86 10.65H7.3a3.2 3.2 0 0 0 2.84-2.07l1.67-5.16c.36-1.13-.3-2.07-1.46-2.07H4.91Zm.4 2.07L3.57 8.47h3.57l.36-1.12H5.4l.28-.91h1.75l.4-1.1H6.07l.3-.83h2l.36-1.1H5.27h.04Z" fill="currentColor"></path></svg>
						Enterprise</a>
				</li>

		<li><a class="group flex items-center px-2 py-0.5 dark:text-gray-300 dark:hover:text-gray-100" href="/pricing">Pricing
			</a></li>

		<li><div class="relative group">
	<button class="px-2 py-0.5 hover:text-gray-500 dark:hover:text-gray-600 flex items-center " type="button">
		<svg class=" text-gray-500 w-5 group-hover:text-gray-400 dark:text-gray-300 dark:group-hover:text-gray-100" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" aria-hidden="true" focusable="false" role="img" width="1em" height="1em" viewBox="0 0 32 18" preserveAspectRatio="xMidYMid meet"><path fill-rule="evenodd" clip-rule="evenodd" d="M14.4504 3.30221C14.4504 2.836 14.8284 2.45807 15.2946 2.45807H28.4933C28.9595 2.45807 29.3374 2.836 29.3374 3.30221C29.3374 3.76842 28.9595 4.14635 28.4933 4.14635H15.2946C14.8284 4.14635 14.4504 3.76842 14.4504 3.30221Z" fill="currentColor"></path><path fill-rule="evenodd" clip-rule="evenodd" d="M14.4504 9.00002C14.4504 8.53382 14.8284 8.15588 15.2946 8.15588H28.4933C28.9595 8.15588 29.3374 8.53382 29.3374 9.00002C29.3374 9.46623 28.9595 9.84417 28.4933 9.84417H15.2946C14.8284 9.84417 14.4504 9.46623 14.4504 9.00002Z" fill="currentColor"></path><path fill-rule="evenodd" clip-rule="evenodd" d="M14.4504 14.6978C14.4504 14.2316 14.8284 13.8537 15.2946 13.8537H28.4933C28.9595 13.8537 29.3374 14.2316 29.3374 14.6978C29.3374 15.164 28.9595 15.542 28.4933 15.542H15.2946C14.8284 15.542 14.4504 15.164 14.4504 14.6978Z" fill="currentColor"></path><path fill-rule="evenodd" clip-rule="evenodd" d="M1.94549 6.87377C2.27514 6.54411 2.80962 6.54411 3.13928 6.87377L6.23458 9.96907L9.32988 6.87377C9.65954 6.54411 10.194 6.54411 10.5237 6.87377C10.8533 7.20343 10.8533 7.73791 10.5237 8.06756L6.23458 12.3567L1.94549 8.06756C1.61583 7.73791 1.61583 7.20343 1.94549 6.87377Z" fill="currentColor"></path></svg>
			
		</button>
	
	
	</div></li>
		<li><hr class="h-5 w-0.5 border-none bg-gray-100 dark:bg-gray-800"></li>
		<li><a class="block cursor-pointer whitespace-nowrap px-2 py-0.5 hover:text-gray-500 dark:text-gray-300 dark:hover:text-gray-100" href="/login">Log In
				</a></li>
			<li><a class="whitespace-nowrap rounded-full border border-transparent bg-gray-900 px-3 py-1 leading-none text-white hover:border-black hover:bg-white hover:text-black" href="/join">Sign Up
					</a></li></ul></nav></div></header></div>
	
	
	
	<div class="SVELTE_HYDRATER contents" data-target="SSOBanner" data-props="{}"></div>
	

	<main class="flex flex-1 flex-col"><div class="flex-1"><div class="container pb-32 pt-28 text-center 2xl:pb-40 2xl:pt-32"><img class="mx-auto mb-4 w-28" src="/front/assets/huggingface_logo_unhappy.svg" alt="">
				<div><h1 class="mx-auto max-w-xl text-4xl font-bold text-gray-800 md:text-6xl">410</h1>
					<p class="mx-auto mt-6 max-w-sm whitespace-pre-line text-lg text-gray-500"><!-- HTML_TAG_START -->https://api-inference.huggingface.co is no longer supported. Please use https://router.huggingface.co instead.<!-- HTML_TAG_END --></p>
					</div>
				</div></div></main>

	<footer class="border-t border-gray-100"><div class="container pb-32 pt-12"><div class="grid gap-8 sm:grid-cols-2 md:grid-cols-5"><div class="sm:col-span-2 md:col-span-1"><div class="SVELTE_HYDRATER contents" data-target="ThemeSwitcher" data-props="{&quot;theme&quot;:&quot;system&quot;,&quot;isLoggedIn&quot;:false}">
<div class="relative inline-block ">
	<button class="rounded-full border border-gray-100 pl-2 py-1 pr-2.5  flex items-center text-sm text-gray-500 bg-white hover:bg-purple-50 hover:border-purple-200 dark:hover:bg-gray-800 dark:hover:border-gray-950 dark:border-gray-800 " type="button">
		<svg class="mr-1.5 text-gray-500" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" fill="currentColor" focusable="false" role="img" width="1em" height="1em" preserveAspectRatio="xMidYMid meet" viewBox="0 0 32 32"><path d="M29 25H3a1 1 0 1 0 0 2h26a1 1 0 1 0 0-2Z" fill="currentColor"></path><path fill-rule="evenodd" clip-rule="evenodd" d="M6 22.5h20a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v13.5a2 2 0 0 0 2 2ZM7 7a1 1 0 0 0-1 1v11a1 1 0 0 0 1 1h18a1 1 0 0 0 1-1V8a1 1 0 0 0-1-1H7Z" fill="currentColor"></path><path d="M6 8a1 1 0 0 1 1-1h18a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V8Z" fill="currentColor" fill-opacity=".4"></path><path d="M29 25H3a1 1 0 1 0 0 2h26a1 1 0 1 0 0-2Z" fill="currentColor"></path></svg>
			System theme
		</button>
	
	
	</div></div></div>
			<div><div class="mb-4 text-lg font-semibold">Website</div>
				<ul class="space-y-1 text-gray-600 md:space-y-2"><li><a class="hover:underline" href="/models">Models </a></li>
					<li><a class="hover:underline" href="/datasets">Datasets </a></li>
					<li><a class="hover:underline" href="/spaces">Spaces </a></li>
					<li class="flex items-center"><a class="group inline-flex gap-x-2.5" href="/changelog"><span class="group-hover:underline">Changelog</span>
							</a></li>
					<li><a class="hover:underline" href="https://endpoints.huggingface.co" target="_blank">Inference Endpoints </a></li>
					<li><a class="hover:underline" href="/chat">HuggingChat </a></li></ul></div>
			<div><div class="mb-4 text-lg font-semibold">Company</div>
				<ul class="space-y-1 text-gray-600 md:space-y-2"><li><a class="hover:underline" href="/huggingface">About </a></li>
					<li><a class="hover:underline" href="/brand">Brand assets </a></li>
					<li><a class="hover:underline" href="/terms-of-service">Terms of service </a></li>
					<li><a class="hover:underline" href="/privacy">Privacy </a></li>
					<li><a class="hover:underline" href="https://apply.workable.com/huggingface/">Careers </a></li>
					<li><a class="hover:underline" href="mailto:press@huggingface.co">Press </a></li></ul></div>
			<div><div class="mb-4 text-lg font-semibold">Resources</div>
				<ul class="space-y-1 text-gray-600 md:space-y-2"><li><a class="hover:underline" href="/learn">Learn </a></li>
					<li><a class="hover:underline" href="/docs">Documentation </a></li>
					<li><a class="hover:underline" href="/blog">Blog </a></li>
					<li><a class="hover:underline" href="https://discuss.huggingface.co">Forum </a></li>
					<li><a class="hover:underline" href="https://status.huggingface.co/">Service Status </a></li></ul></div>
			<div><div class="mb-4 text-lg font-semibold">Social</div>
				<ul class="space-y-1 text-gray-600 md:space-y-2"><li><a class="hover:underline" href="https://github.com/huggingface">GitHub </a></li>
					<li><a class="hover:underline" href="https://twitter.com/huggingface">Twitter </a></li>
					<li><a class="hover:underline" href="https://www.linkedin.com/company/huggingface/">LinkedIn </a></li>
					<li><a class="hover:underline" href="/join/discord">Discord </a></li>
					</ul></div></div></div></footer></div>
		<script>
			 import("\/front\/build\/kube-3c60e60\/index.js"); window.moonSha = "kube-3c60e60\/"; window.__hf_deferred =
			{};
		</script>
		<!-- Stripe -->
		<script>
			if (["hf.co", "huggingface.co"].includes(window.location.hostname)) {
				const script = document.createElement("script");
				script.src = "https://js.stripe.com/v3/";
				script.async = true;
				document.head.appendChild(script);
			}
		</script>
	</body>
</html>


- **Endpoint**: https://api-inference.huggingface.co
- **Attempted at**: 2025-12-23 19:16:24

## Model: nvidia/nemotron-4-340b (FAILED)

**Error**: model does not exist

- **Endpoint**: https://integrate.api.nvidia.com/v1
- **Attempted at**: 2025-12-23 19:16:29

## Model: gpt-4 (FAILED)

**Error**: failed to check model existence: API request failed with status 429: <html>
<head><title>429 Too Many Requests</title></head>
<body>
<center><h1>429 Too Many Requests</h1></center>
<hr><center>nginx</center>
</body>
</html>


- **Endpoint**: https://api.chutes.ai/v1/chat/completions
- **Attempted at**: 2025-12-23 19:16:30

## Model: Qwen/Qwen2-72B-Instruct (FAILED)

**Error**: failed to check model existence: API request failed with status 401: "Api key is invalid"

- **Endpoint**: https://api.siliconflow.cn/v1
- **Attempted at**: 2025-12-23 19:16:30

## Model: moonshot-v1-128k (FAILED)

**Error**: failed to check model existence: API request failed with status 401: {"error":{"message":"Invalid Authentication","type":"invalid_authentication_error"}}

- **Endpoint**: https://api.moonshot.cn/v1
- **Attempted at**: 2025-12-23 19:16:31

## Model: gemini-2.0-flash-exp (FAILED)

**Error**: failed to check model existence: API request failed with status 401: {
  "error": {
    "code": 401,
    "message": "Request had invalid authentication credentials. Expected OAuth 2 access token, login cookie or other valid authentication credential. See https://developers.google.com/identity/sign-in/web/devconsole-project.",
    "status": "UNAUTHENTICATED"
  }
}


- **Endpoint**: https://generativelanguage.googleapis.com/v1
- **Attempted at**: 2025-12-23 19:16:32

## Model: anthropic/claude-3.5-sonnet (FAILED)

**Error**: failed to check model existence: API request failed with status 404: {"error":{"message":"Not Found","code":404}}

- **Endpoint**: https://openrouter.ai/api/v1/chat/completions
- **Attempted at**: 2025-12-23 19:16:33

## Model: zai-large (FAILED)

**Error**: failed to check model existence: API request failed with status 404: <html>
<head><title>404 Not Found</title></head>
<body>
<center><h1>404 Not Found</h1></center>
<hr><center>nginx</center>
</body>
</html>


- **Endpoint**: https://api.z.ai/v1/chat/completions
- **Attempted at**: 2025-12-23 19:16:33

## Model: deepseek-chat

### Basic Information
- **Endpoint**: https://api.deepseek.com
- **Verified at**: 2025-12-23 19:16:34

### Performance Scores
- **Overall Score**: 51.91
- **Code Capability**: 48.00
- **Responsiveness**: 78.00
- **Reliability**: 100.00
- **Feature Richness**: 31.67
- **Value Proposition**: 6.74

### Availability
- **Exists**: true
- **Responsive**: true
- **Overloaded**: false
- **Response Time**: 1.525577251s

### Response Time Metrics
- **Average Latency**: 1.45367503s
- **Throughput**: 5.00 requests/sec

### Supported Features
- **Tool Use**: true
- **Function Calling**: true
- **Code Generation**: true
- **Code Completion**: true
- **Code Explanation**: true
- **Code Review**: false
- **Embeddings**: false
- **Reranking**: false
- **Image Generation**: false
- **Audio Generation**: false
- **Video Generation**: false
- **MCPs**: false
- **LSPs**: false
- **Multimodal**: false
- **Streaming**: false
- **JSON Mode**: false
- **Structured Output**: false
- **Reasoning**: false
- **Parallel Tool Use**: false (Max 0 calls)

### Code Capabilities
- **Language Support**: Python, JavaScript, TypeScript, Java, C++, Go, Rust, C#, PHP, Ruby, Swift, Kotlin, Scala, R, MATLAB, SQL, HTML, CSS, Shell, PowerShell, Dart, Elixir
- **Code Generation**: true
- **Code Completion**: true
- **Code Debugging**: false
- **Code Optimization**: false
- **Code Review**: true
- **Test Generation**: false
- **Documentation**: false
- **Refactoring**: false
- **Error Resolution**: false
- **Architecture Understanding**: false
- **Security Assessment**: false
- **Pattern Recognition**: false
- **Complexity Level**: 0/5
- **Code Quality Score**: 0.00
- **Logic Correctness Score**: 0.00
- **Runtime Efficiency Score**: 0.00

### Language-Specific Performance
- **Python Success Rate**: 66.67%
- **JavaScript Success Rate**: 100.00%
- **Go Success Rate**: 100.00%
- **Java Success Rate**: 100.00%
- **C++ Success Rate**: 100.00%
- **TypeScript Success Rate**: 100.00%
- **Overall Success Rate**: 94.44%

## Category Rankings

### Overall Performance
1. **deepseek-chat**: 51.91

### By Code Capability
1. **deepseek-chat**: 48.00

### By Responsiveness
1. **deepseek-chat**: 78.00

### By Reliability
1. **deepseek-chat**: 100.00

### By Feature Richness
1. **deepseek-chat**: 31.67

### By Value Proposition
1. **deepseek-chat**: 6.74


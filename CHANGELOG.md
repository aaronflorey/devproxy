# Changelog

## [0.2.0](https://github.com/aaronflorey/devproxy/compare/v0.1.0...v0.2.0) (2026-05-19)


### Features

* **01-discovery-domains-and-conflict-policy-01:** bootstrap Go module and CLI root ([b0d6e0d](https://github.com/aaronflorey/devproxy/commit/b0d6e0dbfa5661e8a2a05bc5f03f91530f5a7a60))
* **01-discovery-domains-and-conflict-policy-01:** define shared config and routing contracts ([268a045](https://github.com/aaronflorey/devproxy/commit/268a04545a1f094c903a59f32c718354b4c50932))
* **01-discovery-domains-and-conflict-policy-02:** implement discovery metadata eligibility and port selection ([afc0ab8](https://github.com/aaronflorey/devproxy/commit/afc0ab85d106e5d02f6935ce702f709d7e4d5d43))
* **01-discovery-domains-and-conflict-policy-03:** implement domain generation and override merging ([b6c52d2](https://github.com/aaronflorey/devproxy/commit/b6c52d2af0b8c16a63b8dba12d1fbbc57b0b231f))
* **01-discovery-domains-and-conflict-policy-04:** implement deterministic conflicts and immutable snapshots ([f70ed27](https://github.com/aaronflorey/devproxy/commit/f70ed27e139e4170e3d359b2481ef78ad2c7daca))
* **01-discovery-domains-and-conflict-policy-05:** add status routes doctor and log read models ([6557a60](https://github.com/aaronflorey/devproxy/commit/6557a6034dbf6ab448f4894476049b270c6b974e))
* **01-discovery-domains-and-conflict-policy-05:** implement reconciler and watcher lifecycle ([ae0d848](https://github.com/aaronflorey/devproxy/commit/ae0d848a64a283757f1aab9a4a12ee4bf894086a))
* **02-01:** implement serving-state snapshot and runtime contracts ([569dc92](https://github.com/aaronflorey/devproxy/commit/569dc92b7efa14fd9666e22c399b665f3d93c40b))
* **02-03:** implement managed HTTP proxy request handling ([1089734](https://github.com/aaronflorey/devproxy/commit/10897344b36bfe0eb64aa4ea69d315d182187d79))
* **02-04:** implement cert inventory and mkcert issuance helpers ([567daf0](https://github.com/aaronflorey/devproxy/commit/567daf0fafee5bf1064b123afd563e373833dc5c))
* **02-05:** wire HTTPS listener and network runtime health ([162efca](https://github.com/aaronflorey/devproxy/commit/162efca01dea988cef80f0869e326266267079e4))
* **02-local-dns-proxy-and-https-serving-02:** implement managed-suffix DNS responder and host lookup ([668cfcc](https://github.com/aaronflorey/devproxy/commit/668cfcca1e29bf7bba4175448ce079529e75695a))
* **02:** close routing and HTTPS runtime gaps ([026ea71](https://github.com/aaronflorey/devproxy/commit/026ea7148774b50cdf0c8f3b5b0b8ea734000735))
* **03-01:** implement foreground daemon app and unix admin socket ([82dbf8f](https://github.com/aaronflorey/devproxy/commit/82dbf8f4cf1cc1c9d9d4e3f8304d6b5d9bb5ccb8))
* **03-02:** add thin operator commands over admin socket ([a8a4a3c](https://github.com/aaronflorey/devproxy/commit/a8a4a3c96bdecb98cc7621b7a86bee74c07a1c02))
* **03-02:** implement unix socket admin API client ([1b6c118](https://github.com/aaronflorey/devproxy/commit/1b6c11801b742c17274396c27a4ab81bf0d162cc))
* **03-03:** implement macOS install orchestration and CLI command ([77389a5](https://github.com/aaronflorey/devproxy/commit/77389a5277df2ad6696fd3fe8daca6413a56e728))
* **03-04:** add doctor diagnostics and selective uninstall flow ([f4d1705](https://github.com/aaronflorey/devproxy/commit/f4d1705db99729c1fa1b787a810f412948d74645))
* **03-05:** enforce root preflights for lifecycle operations ([0457f33](https://github.com/aaronflorey/devproxy/commit/0457f33d704c15e98184395142bf197102a00a55))
* **03-06:** gate proxy diagnostics on control-plane runtime health ([082d090](https://github.com/aaronflorey/devproxy/commit/082d09015280ba01054396e63aa7d2128ceeb8c8))
* **03-07:** handle bootout exit-5 missing-service fallback ([7d73576](https://github.com/aaronflorey/devproxy/commit/7d735768dff7b9dc7554f7a283078ef025765d46))
* **04-01:** add daemon-mediated UI control and startup endpoints ([825c56d](https://github.com/aaronflorey/devproxy/commit/825c56d2801c1568fb4ce2a9b36e0f387662067d))
* **04-02:** implement localhost dashboard command and pages ([9f0727b](https://github.com/aaronflorey/devproxy/commit/9f0727b4ce9bb09144b4cd95b89978666948dc95))
* **04-03:** implement menubar runtime and command wiring ([585c24e](https://github.com/aaronflorey/devproxy/commit/585c24e98872ca98466209eefd94d15b1e406ac2))
* **04-04:** surface HTTPS fallback reason in dashboard routes ([c0028ee](https://github.com/aaronflorey/devproxy/commit/c0028eeccead045c1fb0964a5e19be6aa1d4d794))
* **04-05:** wire systray route items to daemon OpenURL actions ([cf760bd](https://github.com/aaronflorey/devproxy/commit/cf760bd9b0336ab7b4394942dba6ba44c6b07139))
* add docs ([f1952df](https://github.com/aaronflorey/devproxy/commit/f1952df5424d302013bce6f7d36c64c25ca73a1f))
* **dashboard:** add navigation, live polling, and new pages ([13a0369](https://github.com/aaronflorey/devproxy/commit/13a03695efb6db2328af8d8ced65a6d5a55300c0))


### Bug Fixes

* **adminapi:** allow group access to admin unix socket ([9d81130](https://github.com/aaronflorey/devproxy/commit/9d811308f2c837f8c857e239822639cc596db437))
* **adminapi:** surface daemon failures and archive v1.0 evidence ([728244f](https://github.com/aaronflorey/devproxy/commit/728244f91158597724a2f814a9fa4188e49dd8d1))
* **daemon:** harden lifecycle diagnostics and routing state ([4a920c8](https://github.com/aaronflorey/devproxy/commit/4a920c86b79d68e30c7dff3a352be6d78411f3fd))
* **daemon:** keep admin API available during degraded startup ([ca71299](https://github.com/aaronflorey/devproxy/commit/ca71299830dc481fd3e704e1ebd586c2a0d5bb29))
* **doctor:** avoid macOS resolver false negatives ([2f8916e](https://github.com/aaronflorey/devproxy/commit/2f8916e4d7e7424df64b31862f50b30e4cfcff00))
* **install:** harden launchd bootstrap diagnostics and idempotency ([1db198d](https://github.com/aaronflorey/devproxy/commit/1db198de5d781824c45dabbd6e6d617790c6f654))
* **install:** stage daemon binary before launchd bootstrap ([68d4d5c](https://github.com/aaronflorey/devproxy/commit/68d4d5c213f1b6bbf02e922c18404f84b0530b0c))
* **install:** target menubar LaunchAgent at active GUI user ([1c44921](https://github.com/aaronflorey/devproxy/commit/1c44921a992366b57e320d8cdc23bb76a0c2e919))
* **launchd:** set explicit PATH and require running daemon state ([aa3b213](https://github.com/aaronflorey/devproxy/commit/aa3b213f58ae71a6b81e7fd1d2d8398bc1af906b))
* **menubar:** keep systray on the main thread ([190bd69](https://github.com/aaronflorey/devproxy/commit/190bd69a5b49d00b5410e538f88a79d167b74d79))
* own admin socket by active macOS GUI user ([1131e6d](https://github.com/aaronflorey/devproxy/commit/1131e6d057a131b0b02917a9969423f68379cff8))
* **runtime:** harden macOS install diagnostics ([57e070a](https://github.com/aaronflorey/devproxy/commit/57e070aae85fbca522b2bafdc9e7397606541bc7))
* **runtime:** refresh certs and harden dashboard flows ([28611df](https://github.com/aaronflorey/devproxy/commit/28611df072c09bcc0ab37fa073a4c695e045e3d0))
* **runtime:** start DNS UDP listener in network runtime ([bb44543](https://github.com/aaronflorey/devproxy/commit/bb44543721090e4598000246d94d4cb3c6f7dae8))
* **startup:** add macOS service controls and menubar install flow ([c0d4b1c](https://github.com/aaronflorey/devproxy/commit/c0d4b1cbc44f78673b340f80fc0525b8b67f4a98))

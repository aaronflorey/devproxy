# Changelog

## [0.2.1](https://github.com/aaronflorey/devproxy/compare/v0.2.0...v0.2.1) (2026-05-20)


### Bug Fixes

* enable CGO ([451ff56](https://github.com/aaronflorey/devproxy/commit/451ff5674e462e4174161b163e69110cc4c7cbbd))

## [0.2.0](https://github.com/aaronflorey/devproxy/compare/v0.1.0...v0.2.0) (2026-05-20)


### Features

* **01-discovery-domains-and-conflict-policy-01:** bootstrap Go module and CLI root ([badbf2a](https://github.com/aaronflorey/devproxy/commit/badbf2ae14323b2426a02402b9e1e25fd1544dbd))
* **01-discovery-domains-and-conflict-policy-01:** define shared config and routing contracts ([a42c318](https://github.com/aaronflorey/devproxy/commit/a42c31848fc5de2c8c9a0879bef50e6ea22bde73))
* **01-discovery-domains-and-conflict-policy-02:** implement discovery metadata eligibility and port selection ([44629e7](https://github.com/aaronflorey/devproxy/commit/44629e72ecc5a301af32ce1cba7f80ab83463481))
* **01-discovery-domains-and-conflict-policy-03:** implement domain generation and override merging ([41127a9](https://github.com/aaronflorey/devproxy/commit/41127a9379d46e446fae76acca2065364512b7de))
* **01-discovery-domains-and-conflict-policy-04:** implement deterministic conflicts and immutable snapshots ([72d219f](https://github.com/aaronflorey/devproxy/commit/72d219f6c847c864c4877d7ab36bc4158af62015))
* **01-discovery-domains-and-conflict-policy-05:** add status routes doctor and log read models ([b24efad](https://github.com/aaronflorey/devproxy/commit/b24efad172c77d2829dac1714d0ede2ab7667cc9))
* **01-discovery-domains-and-conflict-policy-05:** implement reconciler and watcher lifecycle ([ba0db8c](https://github.com/aaronflorey/devproxy/commit/ba0db8c6c0ef008cffc7c859a33cd279b4e90f9f))
* **02-01:** implement serving-state snapshot and runtime contracts ([5380975](https://github.com/aaronflorey/devproxy/commit/53809754337827822eb1a1141ed830e18c21c607))
* **02-03:** implement managed HTTP proxy request handling ([61e9933](https://github.com/aaronflorey/devproxy/commit/61e9933510ea0f356c059a3b73be76ff34072dba))
* **02-04:** implement cert inventory and mkcert issuance helpers ([4059db6](https://github.com/aaronflorey/devproxy/commit/4059db63d9c28dd529b983943b06e34677329a06))
* **02-05:** wire HTTPS listener and network runtime health ([9ea354f](https://github.com/aaronflorey/devproxy/commit/9ea354ff18cf15d79812bf5253dcb718d9634262))
* **02-local-dns-proxy-and-https-serving-02:** implement managed-suffix DNS responder and host lookup ([3b93bd5](https://github.com/aaronflorey/devproxy/commit/3b93bd570aff941132b8dcc6807a61b27cdd9105))
* **02:** close routing and HTTPS runtime gaps ([0cd45f6](https://github.com/aaronflorey/devproxy/commit/0cd45f6551b98f33a4b3ce7af6496c280fb3b043))
* **03-01:** implement foreground daemon app and unix admin socket ([c224b20](https://github.com/aaronflorey/devproxy/commit/c224b20c6e9e01a8d5faf26b341241421f93fe7e))
* **03-02:** add thin operator commands over admin socket ([7d8422d](https://github.com/aaronflorey/devproxy/commit/7d8422da0822f9da6bee11e0845cd034db5d1abb))
* **03-02:** implement unix socket admin API client ([bc22861](https://github.com/aaronflorey/devproxy/commit/bc228612110ea6735da93bf4b7893d81fb870c7c))
* **03-03:** implement macOS install orchestration and CLI command ([886ab1d](https://github.com/aaronflorey/devproxy/commit/886ab1d178cb0d172ad1c8276ab753fa25fded14))
* **03-04:** add doctor diagnostics and selective uninstall flow ([07a0a71](https://github.com/aaronflorey/devproxy/commit/07a0a71997dd03ac5059a1204d3043b258f4b133))
* **03-05:** enforce root preflights for lifecycle operations ([69a7e8f](https://github.com/aaronflorey/devproxy/commit/69a7e8f618eda14ca37494d10ce7af86728682c4))
* **03-06:** gate proxy diagnostics on control-plane runtime health ([4de0199](https://github.com/aaronflorey/devproxy/commit/4de0199201194a7e81048c8b741251bdb37f6ea9))
* **03-07:** handle bootout exit-5 missing-service fallback ([26cb968](https://github.com/aaronflorey/devproxy/commit/26cb968869e9bfa5239fe790bdc90698df8fad79))
* **04-01:** add daemon-mediated UI control and startup endpoints ([8806102](https://github.com/aaronflorey/devproxy/commit/8806102d95ac287fea3e3dc4dbc29c89e4a841c4))
* **04-02:** implement localhost dashboard command and pages ([be30f63](https://github.com/aaronflorey/devproxy/commit/be30f63a43135f1c6364cf04a03e27c146f4cde3))
* **04-03:** implement menubar runtime and command wiring ([6d8bc65](https://github.com/aaronflorey/devproxy/commit/6d8bc65a5c13c456b4cab74a53649fb84ae151c1))
* **04-04:** surface HTTPS fallback reason in dashboard routes ([38a39e6](https://github.com/aaronflorey/devproxy/commit/38a39e673c713516f16e7d6fe61685387fb5d47f))
* **04-05:** wire systray route items to daemon OpenURL actions ([2821daa](https://github.com/aaronflorey/devproxy/commit/2821daa6b97e01c505faa0bc7fc08058bb34c4ec))
* add docs ([6fada8f](https://github.com/aaronflorey/devproxy/commit/6fada8feb4410b35022aca37269ffc68d342ef3e))
* **dashboard:** add navigation, live polling, and new pages ([b1a1282](https://github.com/aaronflorey/devproxy/commit/b1a1282ad70d201a8730480cb23cc8ce7b2b6099))


### Bug Fixes

* **adminapi:** allow group access to admin unix socket ([6c59916](https://github.com/aaronflorey/devproxy/commit/6c599167c9b22d902473a6841c19c6e32073756b))
* **adminapi:** surface daemon failures and archive v1.0 evidence ([a9c09c6](https://github.com/aaronflorey/devproxy/commit/a9c09c6c771d30da84faabed2b187aa2f5d0538e))
* **daemon:** harden lifecycle diagnostics and routing state ([3a536d9](https://github.com/aaronflorey/devproxy/commit/3a536d9574912f84c3c4f0340c45da3067a5cbda))
* **daemon:** keep admin API available during degraded startup ([e253bd0](https://github.com/aaronflorey/devproxy/commit/e253bd0a629fccf07d7060d744ecec2ae6f3abfe))
* **doctor:** avoid macOS resolver false negatives ([06d60df](https://github.com/aaronflorey/devproxy/commit/06d60df412f264c0457aeeebc5dfc9972976d2da))
* **install:** harden launchd bootstrap diagnostics and idempotency ([2c33083](https://github.com/aaronflorey/devproxy/commit/2c330836c13a1a20a5c0e3f7ca7860ec44f79426))
* **install:** stage daemon binary before launchd bootstrap ([b54883b](https://github.com/aaronflorey/devproxy/commit/b54883b17c29d3ca23a28eef87ea4ba2bc7c5ecd))
* **install:** target menubar LaunchAgent at active GUI user ([b5c2a9a](https://github.com/aaronflorey/devproxy/commit/b5c2a9ab57295607246e10ca377709058a81bd5d))
* **launchd:** set explicit PATH and require running daemon state ([d17833f](https://github.com/aaronflorey/devproxy/commit/d17833f943c2dd6f9c831f6a1d7b636452d288ee))
* **menubar:** keep systray on the main thread ([9f8262e](https://github.com/aaronflorey/devproxy/commit/9f8262e8710bad09351b2c511de655a251c6e52f))
* own admin socket by active macOS GUI user ([0f40d57](https://github.com/aaronflorey/devproxy/commit/0f40d57bdfe0d71835b5ad2af32b30f63b5de357))
* **runtime:** harden macOS install diagnostics ([bf63d87](https://github.com/aaronflorey/devproxy/commit/bf63d877cbd62bba1588c19200f0da5cad67f9b6))
* **runtime:** refresh certs and harden dashboard flows ([8cb7d65](https://github.com/aaronflorey/devproxy/commit/8cb7d65ef5c760df0fe9003f916d2092174d9ed2))
* **runtime:** start DNS UDP listener in network runtime ([d90c90b](https://github.com/aaronflorey/devproxy/commit/d90c90b718c96339742cd9cd48d47f4c1632c1fd))
* **startup:** add macOS service controls and menubar install flow ([ff54a63](https://github.com/aaronflorey/devproxy/commit/ff54a636b8ed1a67653f998de1ea1107dfcc3251))
* **tests:** stabilize admin socket test on macos ([32c4d4c](https://github.com/aaronflorey/devproxy/commit/32c4d4cc067f16b632d5882f313106792e5397bd))
* **tests:** stabilize macOS socket and launchd checks ([7b7f01e](https://github.com/aaronflorey/devproxy/commit/7b7f01efbce2a34d8ad7df40152f6554c54db34c))

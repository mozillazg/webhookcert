# ChangeLog

## [0.4.1] (2022-01-13)

* Fix cpu busy bug

## [0.4.0] (2021-12-25)

* Add CheckServerCertValid to check whether server is using latest certs
* Add Organizations and RSAKeySize
* Add WatchAndEnsureWebhooksCA to watch and patch ca for webhook

## [0.3.0] (2021-12-17)

* Change default value of `CertValidityDuration` to 100 years
* Add more tests

## [0.2.0] (2021-10-30)

* Add `CertOption.Hosts` to instead of `CertOption.DNSNames`
* No longer export some needless functions and structs


## 0.1.0 (2021-10-11)

* Initial Release


[0.2.0]: https://github.com/mozillazg/webhookcert/compare/v0.1.0...v0.2.0
[0.3.0]: https://github.com/mozillazg/webhookcert/compare/v0.2.0...v0.3.0
[0.4.0]: https://github.com/mozillazg/webhookcert/compare/v0.3.0...v0.4.0
[0.4.1]: https://github.com/mozillazg/webhookcert/compare/v0.4.0...v0.4.1
[0.5.0]: https://github.com/mozillazg/webhookcert/compare/v0.4.1...v0.5.0
[0.6.0]: https://github.com/mozillazg/webhookcert/compare/v0.5.0...v0.6.0

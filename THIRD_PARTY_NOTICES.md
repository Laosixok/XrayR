# XrayR 1.0.0 protocol runtime sources

The SSPanel-Metron adapter uses the AnyTLS and Hysteria2 services from
[Mtoly/XrayRP](https://github.com/Mtoly/XrayRP/tree/4a54ac6e91242952dbc7156c0f6e748301952c42),
pinned in go.mod to commit 4a54ac6e91242952dbc7156c0f6e748301952c42.
XrayRP source is covered by its Mozilla Public License 2.0.

AnyTLS runtime dependencies include [sing-box](https://github.com/SagerNet/sing-box)
and [sing-anytls](https://github.com/anytls/sing-anytls), covered by their GNU GPL
version 3 or later licenses. See the pinned modules' LICENSE files for the full
terms and copyright notices. go.mod and go.sum identify the dependency versions
used to build this project. The original project's LICENSE covers its own source;
it does not replace dependency licenses.

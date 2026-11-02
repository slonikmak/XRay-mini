# Bundled Xray-core notice

Xray Mini distributes the unmodified official **Xray-core v26.10.29**, Windows
x64, from <https://github.com/XTLS/Xray-core/releases/tag/v26.10.29>.
Xray-core is a separate executable, licensed under the Mozilla Public License
2.0. The complete license accompanies the package as `Xray-core-LICENSE`.
Source for this exact release, including its third-party source/dependency
notices, is available at <https://github.com/XTLS/Xray-core/tree/v26.10.29> and
<https://github.com/XTLS/Xray-core/archive/refs/tags/v26.10.29.zip>.
No Xray-core source modifications are included in this distribution.

Only `xray.exe` and the license are extracted from the official runtime archive.
Wintun, launch scripts, GeoIP and GeoSite data are not shipped or needed by the
restricted MVP configuration. This app does not enable TUN or require Wintun.

Official ZIP SHA-256:
`fce33e81cccf50e7aab572b175519a50b5e0f9d2f5c8d1ed73012ea5c9ad4d58`

Exact extracted `xray.exe` SHA-256:
`07a332b714a3c0eee047a22a9c0ea749b64df33e4e1ff6d28a8c7f9ef943ce10`

Pins established 2026-10-30 by comparing the downloaded official archive to the
GitHub release API asset SHA-256, then hashing the extracted executable. Hash
verification is not a claim that transport compatibility or Windows runtime
acceptance has passed. Run the documented Windows/network acceptance tests.

Build/package with `python3 scripts/package_runtime.py bin`; place `runtime/`
next to `XrayMini.exe`. The app checks the exact executable digest before each
preflight or working invocation. Updating Xray requires a reviewed pin update,
license review, rebuild and repeated acceptance tests; no automatic updates.

The installation directory should not be writable by other users. Digest
checking detects substitution at verification time; it is not a sandbox against
malware already controlling the same Windows user and modifying files between
verification and process creation.

# Bundled Inter font

Inter Variable v4.1 is distributed unmodified and embedded in the application.
Font source: https://github.com/rsms/inter/blob/v4.1/docs/font-files/InterVariable.woff2
License source: https://github.com/rsms/inter/blob/v4.1/LICENSE.txt
Font SHA-256: 693b77d4f32ee9b8bfc995589b5fad5e99adf2832738661f5402f9978429a8e3

Copyright (c) 2016 The Inter Project Authors (https://github.com/rsms/inter)

This Font Software is licensed under the SIL Open Font License, Version 1.1.
This license is copied below, and is also available with a FAQ at:
http://scripts.sil.org/OFL

-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION AND CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.

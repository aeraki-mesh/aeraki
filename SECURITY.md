<!--
# Copyright Aeraki Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
-->

# Security Policy for Aeraki

Version: **v1.0 (2022-06-17)**

## Preface

As a deployment tool, Aeraki needs to have production access which makes
security a very important topic. The Aeraki-Mesh team takes security very
seriously and is continuously working on improving it.

## A word about security scanners

Many organisations these days employ security scanners to validate their
container images before letting them on their clusters, and that is a good
thing. However, the quality and results of these scanners vary greatly,
many of them produce false positives and require people to look at the
issues reported and validate them for correctness. A great example of that
is, that some scanners report kernel vulnerabilities for container images
just because they are derived from some distribution.

We kindly ask you to not raise issues or contact us regarding any issues
that are found by your security scanner. Many of those produce a lot of false
positives, and many of these issues don't affect Aeraki. We do have scanners
in place for our code, dependencies and container images that we publish. We
are well aware of the issues that may affect Aeraki and are constantly
working on the remediation of those that affect Aeraki and our users.

If you believe that we might have missed an issue that we should take a look
at (that can happen), then please discuss it with us. If there is a CVE
assigned to the issue, please do open an issue on our GitHub tracker instead
of writing to the security contact e-mail, since things reported by scanners
are public already and the discussion that might emerge is of benefit to the
general community. However, please validate your scanner results and its
impact on Aeraki before opening an issue at least roughly.

## Supported Versions

We currently support the most recent release (`N`, e.g. `1.8`) and the release
previous to the most recent one (`N-1`, e.g. `1.7`). With the release of
`N+1`, `N-1` drops out of support and `N` becomes `N-1`.

We regularly perform patch releases (e.g. `1.8.5` and `1.7.12`) for the
supported versions, which will contain fixes for security vulnerabilities and
important bugs. Prior releases might receive critical security fixes on a best
effort basis, however, it cannot be guaranteed that security fixes get
back-ported to these unsupported versions.

In rare cases, where a security fix needs complex re-design of a feature or is
otherwise very intrusive, and there's a workaround available, we may decide to
provide a forward-fix only, e.g. to be released the next minor release, instead
of releasing it within a patch branch for the currently supported releases.

## Reporting a Vulnerability

If you find a security related bug in Aeraki, we kindly ask you for responsible
disclosure and for giving us appropriate time to react, analyze and develop a
fix to mitigate the found security vulnerability.

Please use the below process to report a vulnerability to the project:

Email:

1. Email the **zhaohuabing@gmail.com**
    * Emails should contain:
        * description of the problem
        * precise and detailed steps (include screenshots) that created the
          problem
        * the affected version(s)
        * any possible mitigations, if known
1. You will receive a reply from one of the maintainers within **7**
   acknowledging receipt of the email.
1. You may be contacted by **zhaohuabing** to further discuss the reported item.
   Please bear with us as we seek to understand the breadth and scope of the
   reported problem, recreate it, and confirm if there is a vulnerability
   present.

We will do our best to react quickly on your inquiry, and to coordinate a fix
and disclosure with you. Sometimes, it might take a little longer for us to
react (e.g. out of office conditions), so please bear with us in these cases.

We will publish security advisories using the
[Git Hub Security Advisories](https://github.com/aeraki-mesh/aeraki/security/advisories)
feature to keep our community well informed, and will credit you for your
findings (unless you prefer to stay anonymous, of course).


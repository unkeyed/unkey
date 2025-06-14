Name:           metald
Version:        %{_version}
Release:        %{_release}
Summary:        VM Management Service for Firecracker and Cloud Hypervisor

License:        Apache-2.0
URL:            https://github.com/unkeyed/unkey
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros
Requires:       firecracker
Requires:       jailer

%description
Metald is a high-performance VM management service providing ConnectRPC APIs 
for Firecracker and Cloud Hypervisor backends with built-in billing metrics collection.

%prep
%autosetup

%build
cd go/deploy/metald
make build

%install
install -D -m 0755 go/deploy/metald/build/metald %{buildroot}%{_bindir}/metald
install -D -m 0644 go/deploy/metald/contrib/systemd/metald.service %{buildroot}%{_unitdir}/metald.service
install -d -m 0755 %{buildroot}/opt/metald
install -d -m 0755 %{buildroot}/opt/metald/sockets
install -d -m 0755 %{buildroot}/opt/metald/logs
install -d -m 0755 %{buildroot}/opt/metald/data
install -d -m 0755 %{buildroot}/srv/jailer
install -d -m 0755 %{buildroot}/opt/vm-assets

%pre
getent group metald >/dev/null || groupadd -r metald
getent passwd metald >/dev/null || useradd -r -g metald -d /opt/metald -s /sbin/nologin -c "Metald VM Management Service" metald

%post
%systemd_post metald.service

%preun
%systemd_preun metald.service

%postun
%systemd_postun_with_restart metald.service

%files
%{_bindir}/metald
%{_unitdir}/metald.service
%dir %attr(0755, metald, metald) /opt/metald
%dir %attr(0755, metald, metald) /opt/metald/sockets
%dir %attr(0755, metald, metald) /opt/metald/logs
%dir %attr(0755, metald, metald) /opt/metald/data
%dir %attr(0755, metald, metald) /srv/jailer
%dir %attr(0755, metald, metald) /opt/vm-assets

%changelog
* Sat Jun 14 2025 Package Maintainer <maintainer@example.com> - 0.1.0-1
- Initial package release
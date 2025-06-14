Name:           billaged
Version:        %{_version}
Release:        %{_release}
Summary:        Billing and Usage Tracking Service

License:        Apache-2.0
URL:            https://github.com/unkeyed/unkey
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros

%description
Billaged is a high-performance billing service that collects and processes
usage metrics from VM management systems with millisecond precision.

%prep
%autosetup

%build
cd go/deploy/billaged
make build

%install
install -D -m 0755 go/deploy/billaged/build/billaged %{buildroot}%{_bindir}/billaged
install -D -m 0644 go/deploy/billaged/contrib/systemd/billaged.service %{buildroot}%{_unitdir}/billaged.service
install -d -m 0755 %{buildroot}/opt/billaged
install -d -m 0755 %{buildroot}/opt/billaged/data
install -d -m 0755 %{buildroot}/opt/billaged/logs

%pre
getent group billaged >/dev/null || groupadd -r billaged
getent passwd billaged >/dev/null || useradd -r -g billaged -d /opt/billaged -s /sbin/nologin -c "Billaged Service" billaged

%post
%systemd_post billaged.service

%preun
%systemd_preun billaged.service

%postun
%systemd_postun_with_restart billaged.service

%files
%{_bindir}/billaged
%{_unitdir}/billaged.service
%dir %attr(0755, billaged, billaged) /opt/billaged
%dir %attr(0755, billaged, billaged) /opt/billaged/data
%dir %attr(0755, billaged, billaged) /opt/billaged/logs

%changelog
* Sat Jun 14 2025 Package Maintainer <maintainer@example.com> - %{_version}-1
- Initial package release
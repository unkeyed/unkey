Name:           builderd
Version:        0.1.0
Release:        1%{?dist}
Summary:        Builderd Multi-Tenant Build Service

License:        Proprietary
URL:            https://github.com/unkeyed/unkey/go/deploy/builderd
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros
Requires:       systemd
Requires(pre):  shadow-utils
Recommends:     docker

# Disable debug package generation for Go binaries
%global debug_package %{nil}
%global _dwz_low_mem_die_limit 0

%description
Builderd is a multi-tenant build service that converts various source types
(Docker images, Git repositories, archives) into optimized rootfs images for
microVM execution with comprehensive observability and tenant isolation.

%prep
%autosetup

%build
make build

%install
# Create directories
install -d %{buildroot}%{_bindir}
install -d %{buildroot}%{_unitdir}
install -d %{buildroot}%{_localstatedir}/log/builderd
install -d %{buildroot}/opt/builderd
install -d %{buildroot}/opt/builderd/scratch
install -d %{buildroot}/opt/builderd/rootfs
install -d %{buildroot}/opt/builderd/workspace
install -d %{buildroot}/opt/builderd/data

# Install binary
install -m 755 build/builderd %{buildroot}%{_bindir}/builderd

# Install systemd service file
install -m 644 contrib/systemd/builderd.service %{buildroot}%{_unitdir}/builderd.service

%pre
getent group builderd >/dev/null || groupadd -r builderd
getent passwd builderd >/dev/null || \
    useradd -r -g builderd -d /opt/builderd -s /sbin/nologin \
    -c "Builderd Multi-Tenant Build Service" builderd
exit 0

%post
%systemd_post builderd.service

%preun
%systemd_preun builderd.service

%postun
%systemd_postun_with_restart builderd.service

%files
%{_bindir}/builderd
%{_unitdir}/builderd.service
%attr(0755,builderd,builderd) %dir /opt/builderd
%attr(0755,builderd,builderd) %dir /opt/builderd/scratch
%attr(0755,builderd,builderd) %dir /opt/builderd/rootfs
%attr(0755,builderd,builderd) %dir /opt/builderd/workspace
%attr(0755,builderd,builderd) %dir /opt/builderd/data
%attr(0755,builderd,builderd) %dir %{_localstatedir}/log/builderd

%changelog
* Fri Jun 13 2025 Unkey Team <dev@unkey.dev> - 0.1.0-1
- Initial RPM package for builderd
- Multi-tenant build service with Docker extraction
- OpenTelemetry observability integration
- Systemd service configuration
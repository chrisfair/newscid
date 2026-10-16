Name:           newscid
Version:        1.3.0
Release:        1%{?dist}
Summary:        Modern Chess Information Database (Scid vs. PC Rewrite)

License:        GPL-3.0-or-later
URL:            https://github.com/chrisfair/newscid
Source0:        https://github.com/chrisfair/newscid/archive/refs/tags/v%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  nodejs >= 18
BuildRequires:  npm
BuildRequires:  python3-pillow
BuildRequires:  gtk3-devel
BuildRequires:  webkit2gtk4.1-devel

Requires:       gtk3
Requires:       webkit2gtk4.1
Requires:       hicolor-icon-theme
Requires:       shared-mime-info
Recommends:     stockfish

%description
NewScid is a fresh, database-compatible rewrite of Scid vs. PC featuring a modern,
intuitive user interface built with Wails v2 (Go backend) and Vue 3 (TypeScript + Vite).

Features include:
- Native binary support for .si4, .sn4, and .sg4 databases
- PGN parser and exporter with full variations and annotation support
- Opening Tree Explorer dynamically calculating move frequencies and score percentages
- Real-time UCI chess engine analysis with evaluation graphs and blunder alerts
- Opponent and Persona Opening Exploiter with custom Polyglot .bin book generation
- Tactics challenge ladder, Blunder Vault, and automated database puzzle miner
- 17 piece vector sets, customizable board themes, and dark/light UI modes

%prep
%autosetup -n newscid-%{version}

%build
make build

%install
make install DESTDIR=%{buildroot} PREFIX=/usr

%post
if [ $1 -eq 1 ]; then
    update-desktop-database /usr/share/applications &>/dev/null || :
    update-mime-database /usr/share/mime &>/dev/null || :
    gtk-update-icon-cache -qtf /usr/share/icons/hicolor &>/dev/null || :
fi

%postun
if [ $1 -eq 0 ]; then
    update-desktop-database /usr/share/applications &>/dev/null || :
    update-mime-database /usr/share/mime &>/dev/null || :
    gtk-update-icon-cache -qtf /usr/share/icons/hicolor &>/dev/null || :
fi

%files
%license LICENSE
%doc README.md
%{_bindir}/newscid
%{_datadir}/applications/newscid.desktop
%{_datadir}/metainfo/org.newscid.NewScid.metainfo.xml
%{_datadir}/mime/packages/newscid.xml
%{_datadir}/icons/hicolor/*/apps/newscid.png
%{_mandir}/man1/newscid.1*

%changelog
* Wed Oct 15 2026 Christopher Patrick Fair <christopherpatrickfair@gmail.com> - 1.2.0-1
- Release v1.2.0: Opponent Opening Exploiter & Custom Polyglot Book Generator

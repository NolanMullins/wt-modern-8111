#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

#define AppName "WT Modern 8111"
#define AppPublisher "Nolan Mullins"
#define AppExeName "wt-modern.exe"

[Setup]
AppId={{46FBC84D-DCE1-4B2D-AE11-8111B56C5985}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL=https://github.com/NolanMullins/wt-modern-8111
AppUpdatesURL=https://github.com/NolanMullins/wt-modern-8111/releases
DefaultDirName={localappdata}\Programs\WT Modern 8111
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename=wt-modern-setup
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
WizardStyle=modern
UninstallDisplayIcon={app}\{#AppExeName}
CloseApplications=yes
RestartApplications=no

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"; Flags: unchecked
Name: "autostart"; Description: "Start WT Modern 8111 with Windows"; GroupDescription: "Startup:"; Flags: unchecked

[Files]
Source: "..\dist\wt-modern-windows-amd64.exe"; DestDir: "{app}"; DestName: "{#AppExeName}"; Flags: ignoreversion

[Icons]
Name: "{group}\WT Modern 8111"; Filename: "{app}\{#AppExeName}"
Name: "{autodesktop}\WT Modern 8111"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Registry]
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "WT Modern 8111"; ValueData: """{app}\{#AppExeName}"" -open=false"; Tasks: autostart; Flags: uninsdeletevalue

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Start WT Modern 8111"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: files; Name: "{app}\wt-modern.exe.old"
Type: files; Name: "{app}\wt-modern.exe.new"

[Code]
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
    RegDeleteValue(
      HKCU,
      'Software\Microsoft\Windows\CurrentVersion\Run',
      'WT Modern 8111'
    );
end;

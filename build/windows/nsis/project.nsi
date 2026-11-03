Unicode true

####
## TeamAgent Orchestrator installer.
##
## Built by `wails3 task windows:package` (see build/windows/Taskfile.yml), which passes the binary:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\..\bin\TeamAgent.exe project.nsi
## The default is a per-user install (no administrator prompt) in %LOCALAPPDATA%\Programs; build with
## INSTALL_SCOPE=machine for Program Files.
##
## Product name, version and company come from "wails_tools.nsh", generated from build/config.yml.
## The installer speaks English and Brazilian Portuguese, like the app.
####
!include "wails_tools.nsh"
!include "StrFunc.nsh"
${StrStr}
${UnStrStr}

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI2.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING

# Remember the language picked in the first dialog (used again by the uninstaller).
!define MUI_LANGDLL_REGISTRY_ROOT "SHCTX"
!define MUI_LANGDLL_REGISTRY_KEY "Software\${INFO_PRODUCTNAME}"
!define MUI_LANGDLL_REGISTRY_VALUENAME "InstallerLanguage"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!if "${WAILS_INSTALL_SCOPE}" == "user"
    # Only in a per-user install: an elevated installer would start the app as administrator.
    !define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!endif
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "PortugueseBR"
!insertmacro MUI_RESERVEFILE_LANGDLL

LangString AppRunning ${LANG_ENGLISH} "${INFO_PRODUCTNAME} is running. Close it and press Retry."
LangString AppRunning ${LANG_PORTUGUESEBR} "O ${INFO_PRODUCTNAME} está aberto. Feche-o e clique em Repetir."
LangString KeepData ${LANG_ENGLISH} "Your settings (%APPDATA%\TeamAgentOrchestrator) and your projects are kept."
LangString KeepData ${LANG_PORTUGUESEBR} "Suas configurações (%APPDATA%\TeamAgentOrchestrator) e seus projetos foram mantidos."

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
    InstallDirRegKey HKCU "Software\${INFO_PRODUCTNAME}" "InstallDir"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
    InstallDirRegKey HKLM "Software\${INFO_PRODUCTNAME}" "InstallDir"
!endif
ShowInstDetails show
ShowUninstDetails show

# Files of a running app cannot be replaced: wait until the user closes it.
!macro waitAppClosed
    retry:
    nsExec::ExecToStack 'tasklist /FI "IMAGENAME eq ${PRODUCT_EXECUTABLE}" /FO CSV /NH'
    Pop $0
    Pop $1
    !ifdef __UNINSTALL__
        ${UnStrStr} $2 $1 "${PRODUCT_EXECUTABLE}"
    !else
        ${StrStr} $2 $1 "${PRODUCT_EXECUTABLE}"
    !endif
    ${If} $2 != ""
        MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "$(AppRunning)" /SD IDCANCEL IDRETRY retry
        Abort
    ${EndIf}
!macroend

Function .onInit
    !insertmacro wails.checkArchitecture
    !insertmacro wails.setShellContext
    !insertmacro MUI_LANGDLL_DISPLAY
FunctionEnd

Function un.onInit
    !insertmacro wails.setShellContext
    !insertmacro MUI_UNGETLANGUAGE
FunctionEnd

Section
    !insertmacro wails.setShellContext
    !insertmacro waitAppClosed

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    WriteRegStr SHCTX "Software\${INFO_PRODUCTNAME}" "InstallDir" "$INSTDIR"
    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext
    !insertmacro waitAppClosed

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    DeleteRegKey SHCTX "Software\${INFO_PRODUCTNAME}"
    !insertmacro wails.deleteUninstaller
    DetailPrint "$(KeepData)"
SectionEnd

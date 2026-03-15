import * as vscode from 'vscode';

export function getCliPath(): string {
    return vscode.workspace.getConfiguration('trove').get<string>('cliPath', '') || 'trove';
}

export function getVaultPath(): string {
    return vscode.workspace.getConfiguration('trove').get<string>('vaultPath', '');
}

export function getDefaultLanguage(): string {
    return vscode.workspace.getConfiguration('trove').get<string>('defaultLanguage', '');
}

export function getShowNotifications(): boolean {
    return vscode.workspace.getConfiguration('trove').get<boolean>('showNotifications', true);
}

export function getSyncOnSave(): boolean {
    return vscode.workspace.getConfiguration('trove').get<boolean>('syncOnSave', false);
}

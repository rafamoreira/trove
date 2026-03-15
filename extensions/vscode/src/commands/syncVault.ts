import * as vscode from 'vscode';
import * as cli from '../cli';
import { getShowNotifications } from '../config';

export async function syncVault(): Promise<void> {
    try {
        const result = await cli.sync();
        if (getShowNotifications()) {
            const parts: string[] = [];
            if (result.committed) { parts.push('committed'); }
            if (result.pushed) { parts.push('pushed'); }
            const msg = parts.length
                ? `Sync complete: ${parts.join(', ')}`
                : 'Sync complete: nothing to do';
            vscode.window.showInformationMessage(msg);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove sync failed: ${msg}`);
    }
}

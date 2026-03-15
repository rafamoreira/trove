import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';
import * as cli from '../cli';
import { SnippetTreeProvider } from '../tree/snippetTreeProvider';
import { getDefaultLanguage, getShowNotifications } from '../config';

const LANGUAGE_CHOICES = [
    'go', 'javascript', 'typescript', 'python', 'ruby', 'rust',
    'shell', 'sql', 'lua', 'plaintext',
];

const VSCODE_LANG_MAP: Record<string, string> = {
    'go': 'go',
    'javascript': 'javascript',
    'javascriptreact': 'javascript',
    'typescript': 'typescript',
    'typescriptreact': 'typescript',
    'python': 'python',
    'ruby': 'ruby',
    'rust': 'rust',
    'shellscript': 'shell',
    'sql': 'sql',
    'lua': 'lua',
    'plaintext': 'plaintext',
};

const LANG_EXT: Record<string, string> = {
    'go': '.go',
    'javascript': '.js',
    'typescript': '.ts',
    'python': '.py',
    'ruby': '.rb',
    'rust': '.rs',
    'shell': '.sh',
    'sql': '.sql',
    'lua': '.lua',
    'plaintext': '.txt',
};

async function promptForSnippetDetails(
    suggestedLang?: string,
): Promise<{ name: string; lang: string; desc: string; tags: string } | undefined> {
    const name = await vscode.window.showInputBox({
        prompt: 'Snippet name',
        placeHolder: 'my_snippet',
        validateInput: (v) => v.trim() ? null : 'Name is required',
    });
    if (!name) { return undefined; }

    const defaultLang = suggestedLang || getDefaultLanguage() || 'plaintext';
    const lang = await vscode.window.showQuickPick(LANGUAGE_CHOICES, {
        placeHolder: 'Select language',
        canPickMany: false,
    });
    if (!lang) { return undefined; }

    const desc = await vscode.window.showInputBox({
        prompt: 'Description (optional)',
        placeHolder: 'What does this snippet do?',
    }) ?? '';

    const tags = await vscode.window.showInputBox({
        prompt: 'Tags (comma-separated, optional)',
        placeHolder: 'util, http, auth',
    }) ?? '';

    return { name, lang, desc, tags };
}

export async function newSnippet(
    vaultPath: string,
    treeProvider: SnippetTreeProvider,
): Promise<void> {
    const details = await promptForSnippetDetails();
    if (!details) { return; }

    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'trove-'));
    const ext = LANG_EXT[details.lang] || '';
    const tmpFile = path.join(tmpDir, details.name + ext);

    try {
        fs.writeFileSync(tmpFile, '');
        const snippet = await cli.add(tmpFile, details.name, details.lang, details.desc, details.tags);
        treeProvider.refresh();

        const filePath = vscode.Uri.file(`${vaultPath}/${snippet.path}`);
        const doc = await vscode.workspace.openTextDocument(filePath);
        await vscode.window.showTextDocument(doc);

        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Created ${snippet.id}`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    } finally {
        try { fs.rmSync(tmpDir, { recursive: true }); } catch { /* ignore */ }
    }
}

export async function newSnippetFromSelection(
    vaultPath: string,
    treeProvider: SnippetTreeProvider,
): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showErrorMessage('No active editor');
        return;
    }

    const text = editor.selection.isEmpty
        ? editor.document.getText()
        : editor.document.getText(editor.selection);

    if (!text.trim()) {
        vscode.window.showErrorMessage('No text selected');
        return;
    }

    const vscodeLang = editor.document.languageId;
    const suggestedLang = VSCODE_LANG_MAP[vscodeLang];

    const details = await promptForSnippetDetails(suggestedLang);
    if (!details) { return; }

    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'trove-'));
    const ext = LANG_EXT[details.lang] || '';
    const tmpFile = path.join(tmpDir, details.name + ext);

    try {
        fs.writeFileSync(tmpFile, text);
        const snippet = await cli.add(tmpFile, details.name, details.lang, details.desc, details.tags);
        treeProvider.refresh();

        const filePath = vscode.Uri.file(`${vaultPath}/${snippet.path}`);
        const doc = await vscode.workspace.openTextDocument(filePath);
        await vscode.window.showTextDocument(doc);

        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Created ${snippet.id}`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    } finally {
        try { fs.rmSync(tmpDir, { recursive: true }); } catch { /* ignore */ }
    }
}

export async function addFile(
    vaultPath: string,
    treeProvider: SnippetTreeProvider,
): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        vscode.window.showErrorMessage('No active editor');
        return;
    }

    const filePath = editor.document.uri.fsPath;
    const vscodeLang = editor.document.languageId;
    const suggestedLang = VSCODE_LANG_MAP[vscodeLang];

    const details = await promptForSnippetDetails(suggestedLang);
    if (!details) { return; }

    try {
        const snippet = await cli.add(filePath, details.name, details.lang, details.desc, details.tags);
        treeProvider.refresh();

        if (getShowNotifications()) {
            vscode.window.showInformationMessage(`Added ${snippet.id}`);
        }
    } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        vscode.window.showErrorMessage(`Trove: ${msg}`);
    }
}

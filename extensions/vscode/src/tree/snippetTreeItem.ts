import * as vscode from 'vscode';
import { TroveSnippet } from '../types';

export class LanguageTreeItem extends vscode.TreeItem {
    constructor(public readonly language: string, snippetCount: number) {
        super(language, vscode.TreeItemCollapsibleState.Collapsed);
        this.contextValue = 'language';
        this.description = `${snippetCount}`;
        this.iconPath = new vscode.ThemeIcon('symbol-folder');
    }
}

export class SnippetTreeItem extends vscode.TreeItem {
    constructor(public readonly snippet: TroveSnippet) {
        super(snippet.name, vscode.TreeItemCollapsibleState.None);
        this.contextValue = 'snippet';
        this.description = snippet.description || '';
        const tagStr = snippet.tags?.length ? `Tags: ${snippet.tags.join(', ')}` : '';
        const created = snippet.created ? `Created: ${snippet.created}` : '';
        this.tooltip = [snippet.id, snippet.description, tagStr, created]
            .filter(Boolean)
            .join('\n');
        this.iconPath = new vscode.ThemeIcon('file-code');
        this.command = {
            command: 'trove.showSnippet',
            title: 'Show Snippet',
            arguments: [snippet],
        };
    }
}

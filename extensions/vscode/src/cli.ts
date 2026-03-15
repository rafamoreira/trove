import { execFile } from 'child_process';
import * as vscode from 'vscode';
import { getCliPath, getVaultPath } from './config';
import {
    TroveEnvelope,
    TroveSnippet,
    ShowResult,
    SearchResult,
    SyncResult,
    RemoveResult,
    ConfigDisplay,
    StatusResult,
} from './types';

export class TroveCliError extends Error {
    constructor(
        message: string,
        public readonly exitCode: number | null,
        public readonly stderr: string,
    ) {
        super(message);
        this.name = 'TroveCliError';
    }
}

let outputChannel: vscode.OutputChannel | undefined;

function getOutputChannel(): vscode.OutputChannel {
    if (!outputChannel) {
        outputChannel = vscode.window.createOutputChannel('Trove');
    }
    return outputChannel;
}

export function disposeOutputChannel(): void {
    outputChannel?.dispose();
    outputChannel = undefined;
}

function exec<T>(args: string[]): Promise<TroveEnvelope<T>> {
    const cliPath = getCliPath();
    const vaultPath = getVaultPath();

    const fullArgs = ['--json'];
    if (vaultPath) {
        fullArgs.push('--vault', vaultPath);
    }
    fullArgs.push(...args);

    return new Promise((resolve, reject) => {
        execFile(cliPath, fullArgs, { maxBuffer: 10 * 1024 * 1024 }, (error, stdout, stderr) => {
            if (error) {
                reject(new TroveCliError(
                    stderr.trim() || error.message,
                    error.code ? parseInt(String(error.code), 10) : null,
                    stderr,
                ));
                return;
            }

            let envelope: TroveEnvelope<T>;
            try {
                envelope = JSON.parse(stdout);
            } catch {
                reject(new TroveCliError(`Failed to parse CLI output: ${stdout.slice(0, 200)}`, null, ''));
                return;
            }

            if (envelope.warnings?.length) {
                const ch = getOutputChannel();
                for (const w of envelope.warnings) {
                    ch.appendLine(`[warn] ${w.message}${w.path ? ` (${w.path})` : ''}`);
                }
            }

            resolve(envelope);
        });
    });
}

export async function list(lang?: string, tag?: string): Promise<TroveSnippet[]> {
    const args = ['list'];
    if (lang) { args.push('--lang', lang); }
    if (tag) { args.push('--tag', tag); }
    const env = await exec<TroveSnippet[]>(args);
    return env.data;
}

export async function show(selector: string): Promise<ShowResult> {
    const env = await exec<ShowResult>(['show', '--meta', selector]);
    return env.data;
}

export async function search(query: string, lang?: string, tag?: string): Promise<SearchResult[]> {
    const args = ['search', query];
    if (lang) { args.push('--lang', lang); }
    if (tag) { args.push('--tag', tag); }
    const env = await exec<SearchResult[]>(args);
    return env.data;
}

export async function add(
    filePath: string,
    name: string,
    lang: string,
    desc?: string,
    tags?: string,
): Promise<TroveSnippet> {
    const args = ['add', filePath, '--name', name, '--lang', lang];
    if (desc) { args.push('--desc', desc); }
    if (tags) { args.push('--tags', tags); }
    const env = await exec<TroveSnippet>(args);
    return env.data;
}

export async function editMeta(
    selector: string,
    desc?: string,
    tags?: string,
): Promise<TroveSnippet> {
    const args = ['edit', selector];
    if (desc !== undefined) { args.push('--desc', desc); }
    if (tags !== undefined) { args.push('--tags', tags); }
    const env = await exec<TroveSnippet>(args);
    return env.data;
}

export async function remove(selector: string): Promise<RemoveResult> {
    const env = await exec<RemoveResult>(['rm', '--force', selector]);
    return env.data;
}

export async function sync(): Promise<SyncResult> {
    const env = await exec<SyncResult>(['sync']);
    return env.data;
}

export async function status(): Promise<StatusResult> {
    const env = await exec<StatusResult>(['status']);
    return env.data;
}

export async function config(): Promise<ConfigDisplay> {
    const env = await exec<ConfigDisplay>(['config', '--show']);
    return env.data;
}

export async function resolveVaultPath(): Promise<string> {
    const vaultOverride = getVaultPath();
    if (vaultOverride) { return vaultOverride; }
    const cfg = await config();
    return cfg.values['vault_path'] as string;
}

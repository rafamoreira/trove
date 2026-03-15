export interface TroveSnippet {
    id: string;
    name: string;
    language: string;
    path: string;
    meta_path: string;
    description: string;
    tags: string[];
    created: string;
}

export interface TroveWarning {
    code: string;
    message: string;
    path?: string;
}

export interface TroveEnvelope<T> {
    data: T;
    warnings: TroveWarning[];
}

export interface ShowResult {
    snippet: TroveSnippet;
    body: string;
}

export interface SearchMatch {
    line: number;
    context: string;
}

export interface SearchResult {
    snippet: TroveSnippet;
    matches: SearchMatch[];
}

export interface SyncResult {
    committed: boolean;
    pushed: boolean;
}

export interface RemoveResult {
    id: string;
    removed: boolean;
}

export interface ConfigDisplay {
    path: string;
    values: Record<string, unknown>;
    sources: Record<string, string>;
}

export interface StatusResult {
    git_available: boolean;
    git_repo: boolean;
    pending_files: string[];
    last_commit?: string;
}

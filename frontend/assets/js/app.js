'use strict';

const TRANSLATIONS = {
    it: {
        title: 'File Renamer',
        madeWith: 'Sviluppata con il supporto di',
        tabBrowse: 'Sfoglia',
        tabPreview: 'Anteprima',
        selectDir: 'Seleziona directory',
        noDir: 'Nessuna directory selezionata',
        styleLabel: 'Stile',
        styles: {
            kebab: 'Kebab',
            snake: 'Snake',
            camel: 'Camel',
        },
        styleHint: {
            kebab: 'Directory-Name / file-name.txt',
            snake: 'Directory_Name / file_name.txt',
            camel: 'DirectoryName / fileName.txt',
        },
        btnDryRun: 'Dry-Run',
        btnApply: 'Applica Rename',
        applying: 'Applicando...',
        running: 'Analizzando...',
        noChanges: 'Nessuna modifica necessaria.',
        previewTitle: 'Modifiche da applicare',
        dirLabel: 'DIR',
        fileLabel: 'FILE',
        emptyDir: 'La directory è vuota.',
        browseHint: 'Seleziona una directory per visualizzarne il contenuto.',
        previewHint: 'Esegui un dry-run per vedere le modifiche proposte.',
        countItems: n => `${n} element${n !== 1 ? 'i' : 'o'}`,
    },
    en: {
        title: 'File Renamer',
        madeWith: 'Developed with the support of',
        tabBrowse: 'Browse',
        tabPreview: 'Preview',
        selectDir: 'Select directory',
        noDir: 'No directory selected',
        styleLabel: 'Style',
        styles: {
            kebab: 'Kebab',
            snake: 'Snake',
            camel: 'Camel',
        },
        styleHint: {
            kebab: 'Directory-Name / file-name.txt',
            snake: 'Directory_Name / file_name.txt',
            camel: 'DirectoryName / fileName.txt',
        },
        btnDryRun: 'Dry-Run',
        btnApply: 'Apply Rename',
        applying: 'Applying...',
        running: 'Scanning...',
        noChanges: 'No changes needed.',
        previewTitle: 'Changes to apply',
        dirLabel: 'DIR',
        fileLabel: 'FILE',
        emptyDir: 'The directory is empty.',
        browseHint: 'Select a directory to view its contents.',
        previewHint: 'Run a dry-run to see the proposed changes.',
        countItems: n => `${n} item${n !== 1 ? 's' : ''}`,
    },
};

function FileRenamer() {
    return {
        lang: navigator.language.startsWith('it') ? 'it' : 'en',
        get t() {
            return TRANSLATIONS[this.lang];
        },

        toggleLang() {
            this.lang = this.lang === 'it' ? 'en' : 'it';
            window.go.main.App.SetLanguage(this.lang);
        },

        // State
        tab: 'browse',          // 'browse' | 'preview'
        selectedDir: '',
        style: 'kebab',         // 'kebab' | 'snake' | 'camel'
        loading: false,
        loadingApply: false,
        browseEntries: [],      // {name, isDir}
        previewEntries: [],     // RenameEntry[]
        result: null,           // {success, message, count} | null

        async init() { },

        resetPreview() {
            this.previewEntries = [];
            this.result = null;
        },

        //  Directory picker 

        async selectDir() {
            if (!window.go?.main?.App) {
                return;
            }
            const dir = await window.go.main.App.SelectDirectory();
            if (!dir) {
                return;
            }
            this.selectedDir = dir;
            this.resetPreview();
            this.tab = 'browse';
            await this.loadBrowse();
        },

        //  Browse tab 

        async loadBrowse() {
            if (!this.selectedDir || !window.go?.main?.App) {
                return;
            }
            try {
                this.browseEntries = await window.go.main.App.ListDirectory(this.selectedDir);
            } catch (e) {
                this.browseEntries = [];
            }
        },

        //  Dry-run 

        async dryRun() {
            if (!this.selectedDir || !window.go?.main?.App) {
                return;
            }
            this.loading = true;
            this.result = null;
            try {
                const entries = await window.go.main.App.PreviewRenames(
                    this.selectedDir, this.style
                );
                this.previewEntries = entries || [];
                this.tab = 'preview';
            } catch (e) {
                this.result = { success: false, message: String(e) };
            } finally {
                this.loading = false;
            }
        },

        //  Apply rename 

        async applyRename() {
            if (!this.selectedDir || !window.go?.main?.App) {
                return;
            }
            this.loadingApply = true;
            this.result = null;
            try {
                const res = await window.go.main.App.ApplyRenames(
                    this.selectedDir, this.style
                );
                this.result = res;
                if (res.success) {
                    this.previewEntries = [];
                    await this.loadBrowse();
                }
            } catch (e) {
                this.result = { success: false, message: String(e) };
            } finally {
                this.loadingApply = false;
            }
        },
    };
}

export const state = {
  currentProject: localStorage.getItem('etools_current_project') || '',
  currentChatId: null,
  currentIssue: JSON.parse(localStorage.getItem('etools_current_issue') || 'null'),
  expandedProjects: new Set(JSON.parse(localStorage.getItem('etools_expanded') || '[]')),
  editContext: null,
  uploadReplaceContext: null,
};
export function saveExpanded(){ localStorage.setItem('etools_expanded', JSON.stringify([...state.expandedProjects])); }
export function saveCurrentProject(){ if(state.currentProject) localStorage.setItem('etools_current_project', state.currentProject); else localStorage.removeItem('etools_current_project'); }
export function saveCurrentIssue(){ if(state.currentIssue) localStorage.setItem('etools_current_issue', JSON.stringify(state.currentIssue)); else localStorage.removeItem('etools_current_issue'); }

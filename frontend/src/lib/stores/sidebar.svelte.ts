class SidebarStore {
  isOpen = $state<boolean>(true);
  isMobileOpen = $state<boolean>(false);

  constructor() {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('sidebar_open');
      if (saved !== null) {
        this.isOpen = saved === 'true';
      }
    }
  }

  toggle() {
    this.isOpen = !this.isOpen;
    if (typeof window !== 'undefined') {
      localStorage.setItem('sidebar_open', String(this.isOpen));
    }
  }

  toggleMobile() {
    this.isMobileOpen = !this.isMobileOpen;
  }

  closeMobile() {
    this.isMobileOpen = false;
  }
}

export const sidebar = new SidebarStore();

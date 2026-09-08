class SidebarStore {
  isOpen = $state<boolean>(true);
  isMobileOpen = $state<boolean>(false);

  toggle() {
    this.isOpen = !this.isOpen;
  }

  toggleMobile() {
    this.isMobileOpen = !this.isMobileOpen;
  }

  closeMobile() {
    this.isMobileOpen = false;
  }
}

export const sidebar = new SidebarStore();

import { writable } from 'svelte/store';

export const notifications = writable([]);

let nextId = 1;

function addNotification(type, message, timeout) {
  const id = nextId++;
  notifications.update(n => [...n, { id, type, message, timeout }]);
  return id;
}

export function notifySuccess(message) {
  return addNotification('success', message, 3000);
}

export function notifyError(message) {
  return addNotification('error', message, 8000);
}

export function notifyInfo(message) {
  return addNotification('info', message, 3000);
}

export function dismiss(id) {
  notifications.update(n => n.filter(item => item.id !== id));
}

export function dismissLatest() {
  let dismissed = false;
  notifications.update(items => {
    if (items.length === 0) return items;
    dismissed = true;
    return items.slice(0, -1);
  });
  return dismissed;
}

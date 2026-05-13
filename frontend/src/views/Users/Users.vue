<template>
  <div class="p-4 sm:p-6 flex flex-col gap-4">
    <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-800 dark:text-gray-200">User Management</h1>
      <Button v-if="canCreateUsers" @click="openCreateModal" icon="pi pi-plus" label="Add User" class="w-full sm:w-auto" />
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
    </div>

    <div v-else-if="error" class="flex flex-col items-center justify-center py-12">
      <i class="pi pi-exclamation-triangle text-4xl text-red-500 mb-4"></i>
      <p class="text-red-400">{{ error }}</p>
      <Button @click="loadUsers" label="Retry" class="mt-4" />
    </div>

    <div v-else class="glass-card rounded-3xl border border-gray-200 dark:border-white/10 overflow-hidden">
    <DataTable
      :value="users"
      :paginator="users.length > 10"
      :rows="10"
      :rowsPerPageOptions="[10, 25, 50]"
      stripedRows
      responsiveLayout="scroll"
      class="p-datatable-sm"
    >
      <Column field="username" header="Username" sortable>
        <template #body="{ data }">
          <div class="flex items-center gap-2">
             <i class="pi pi-user text-gray-400 dark:text-gray-400"></i>
             <span class="text-gray-800 dark:text-gray-200">{{ data.username }}</span>
          </div>
        </template>
      </Column>
      <Column field="full_name" header="Name" sortable>
        <template #body="{ data }">
           <span class="text-gray-600 dark:text-gray-300">{{ data.full_name || '-' }}</span>
        </template>
      </Column>
      <Column field="email" header="Email" sortable class="hidden sm:table-cell">
        <template #body="{ data }">
           <span class="text-gray-600 dark:text-gray-300">{{ data.email || '-' }}</span>
        </template>
      </Column>
      <Column field="role" header="Role" sortable>
        <template #body="{ data }">
          <Tag :value="data.role" :severity="getRoleSeverity(data.role)" />
        </template>
      </Column>
      <Column header="Permissions">
        <template #body="{ data }">
          <div class="flex flex-wrap gap-1">
            <Tag
              v-for="permission in getRolePermissions(data.role).slice(0, 3)"
              :key="`${data.id}-${permission}`"
              :value="permission"
              severity="secondary"
            />
            <Tag
              v-if="getRolePermissions(data.role).length > 3"
              :value="`+${getRolePermissions(data.role).length - 3}`"
              severity="contrast"
            />
          </div>
        </template>
      </Column>
      <Column field="enabled" header="Status" sortable>
        <template #body="{ data }">
          <Tag
            :value="data.enabled ? 'Active' : 'Inactive'"
            :severity="data.enabled ? 'success' : 'danger'"
          />
        </template>
      </Column>
      <Column header="Expires" sortable field="expires_at">
        <template #body="{ data }">
           <span v-if="data.expires_at" class="text-gray-600 dark:text-gray-300 text-sm">
            {{ formatDate(data.expires_at) }}
          </span>
           <span v-else-if="data.auto_deactivate_days > 0" class="text-yellow-600 dark:text-yellow-400 text-sm">
            After {{ data.auto_deactivate_days }} days
          </span>
           <span v-else class="text-gray-400 dark:text-gray-500 text-sm">Never</span>
        </template>
      </Column>
      <Column header="Actions" :exportable="false">
        <template #body="{ data }">
          <div class="flex gap-2">
            <Button
              v-if="canEditUsers"
              :icon="data.enabled ? 'pi pi-ban' : 'pi pi-check'"
              size="small"
              text
              :severity="data.enabled ? 'warning' : 'success'"
              @click="toggleUserEnabled(data)"
              v-tooltip="data.enabled ? 'Deactivate' : 'Activate'"
            />
            <Button
              v-if="canEditUsers"
              icon="pi pi-pencil"
              size="small"
              text
              @click="editUser(data)"
              v-tooltip="'Edit user'"
            />
            <Button
              v-if="canDeleteUsers"
              icon="pi pi-trash"
              size="small"
              text
              severity="danger"
              @click="confirmDeleteUser(data)"
              v-tooltip="'Delete user'"
            />
          </div>
        </template>
      </Column>
      <template #empty>
         <div class="text-center py-8 text-gray-400 dark:text-gray-500">
          <i class="pi pi-users text-4xl mb-2 block"></i>
          <p>No users found</p>
        </div>
      </template>
     </DataTable>
     </div>

     <Dialog
      v-model:visible="showModal"
      :header="isEditMode ? 'Edit User' : 'Create User'"
      modal
      class="w-full max-w-lg mx-4"
    >
      <div class="flex flex-col gap-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Username *</label>
            <InputText v-model="formData.username" class="w-full" :disabled="isEditMode" placeholder="username" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Full Name *</label>
            <InputText v-model="formData.full_name" class="w-full" placeholder="Max Mustermann" />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Email *</label>
          <InputText v-model="formData.email" type="email" class="w-full" placeholder="user@example.com" />
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Role *</label>
            <Dropdown
              v-model="formData.role"
              :options="roles"
              optionLabel="label"
              optionValue="value"
              class="w-full"
            />
             <small class="text-gray-400 dark:text-gray-500">{{ roleMeta[formData.role]?.description || '' }}</small>
          </div>
          <div v-if="!isEditMode">
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Password *</label>
            <Password v-model="formData.password" :feedback="true" toggleMask class="w-full" />
          </div>
        </div>

        <div v-if="isEditMode" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">New Password</label>
            <Password v-model="formData.password" :feedback="true" toggleMask class="w-full" placeholder="Leave empty to keep" />
             <small class="text-gray-400 dark:text-gray-500">Leave empty to keep current password</small>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Description</label>
            <InputText v-model="formData.description" class="w-full" placeholder="Optional note" />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Auto-Deactivate (days)</label>
            <InputNumber v-model="formData.auto_deactivate_days" :min="0" :max="3650" class="w-full" placeholder="0 = never" />
             <small class="text-gray-400 dark:text-gray-500">0 = no auto-deactivation</small>
          </div>
          <div v-if="formData.expires_at">
            <label class="block text-sm font-medium text-gray-600 dark:text-gray-300 mb-1">Expires At</label>
            <InputText :modelValue="formatDate(formData.expires_at)" class="w-full" disabled />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Checkbox v-model="formData.enabled" binary inputId="enabled-cb" />
           <label for="enabled-cb" class="text-sm text-gray-600 dark:text-gray-300">Enabled</label>
        </div>

         <div class="rounded-2xl border border-gray-200 dark:border-white/10 bg-gray-50 dark:bg-black/20 p-3">
           <div class="text-sm font-medium text-gray-800 dark:text-gray-200 mb-2">Assigned permissions</div>
          <div class="flex flex-wrap gap-2">
            <Tag
              v-for="permission in selectedRolePermissions"
              :key="permission"
              :value="permission"
              severity="info"
            />
             <span v-if="selectedRolePermissions.length === 0" class="text-gray-400 dark:text-gray-500 text-sm">No permissions</span>
          </div>
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" severity="secondary" @click="closeModal" />
        <Button
          v-if="isEditMode ? canEditUsers : canCreateUsers"
          :label="isEditMode ? 'Update' : 'Create'"
          @click="saveUser"
          :loading="saving"
        />
      </template>
    </Dialog>

    <Toast />
    <ConfirmDialog />
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue';
import axios from '../../axios.js';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import InputText from 'primevue/inputtext';
import InputNumber from 'primevue/inputnumber';
import Dropdown from 'primevue/dropdown';
import Password from 'primevue/password';
import Checkbox from 'primevue/checkbox';
import Tag from 'primevue/tag';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';
import { useAuthStore } from '../../stores/auth';

const users = ref([]);
const loading = ref(true);
const error = ref(null);
const saving = ref(false);
const showModal = ref(false);
const isEditMode = ref(false);
const toast = useToast();
const confirm = useConfirm();
const auth = useAuthStore();

const defaultFormData = () => ({
  id: null,
  username: '',
  full_name: '',
  email: '',
  role: 'benutzer',
  password: '',
  enabled: true,
  auto_deactivate_days: 0,
  expires_at: null,
  description: ''
});

const formData = ref(defaultFormData());

const roles = [
  { label: 'Admin - Vollzugriff', value: 'admin' },
  { label: 'Techniker - Proxies anlegen und bearbeiten', value: 'techniker' },
  { label: 'Benutzer - Ansehen, starten/stoppen', value: 'benutzer' },
  { label: 'Auditor - Audit-Logs einsehen', value: 'auditor' }
]

const roleMeta = {
  admin: {
    description: 'Vollständige Administration',
    permissions: ['proxy:*', 'device:*', 'config:*', 'system:*', 'user:*', 'audit:*', 'logs:*']
  },
  techniker: {
    description: 'Proxies anlegen, bearbeiten, löschen; keine Admin-Einstellungen',
    permissions: ['proxy:view', 'proxy:create', 'proxy:edit', 'proxy:delete', 'proxy:control', 'device:view', 'device:edit', 'config:view', 'system:view', 'logs:view']
  },
  benutzer: {
    description: 'Proxies ansehen, starten/stoppen; keine Änderungen',
    permissions: ['proxy:view', 'proxy:control', 'device:view', 'config:view', 'system:view', 'logs:view']
  },
  auditor: {
    description: 'Audit- und Compliance-Einsicht',
    permissions: ['proxy:view', 'device:view', 'config:view', 'system:view', 'audit:view', 'audit:export', 'logs:view', 'logs:export']
  }
}
};

const canCreateUsers = computed(() => auth.hasPermission('user:create'));
const canEditUsers = computed(() => auth.hasPermission('user:edit'));
const canDeleteUsers = computed(() => auth.hasPermission('user:delete'));
const selectedRolePermissions = computed(() => roleMeta[formData.value.role]?.permissions || []);

const getRolePermissions = (role) => roleMeta[role]?.permissions || [];

const formatDate = (dateStr) => {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
};

const loadUsers = async () => {
  loading.value = true;
  error.value = null;
  try {
    const response = await axios.get('/api/users');
    users.value = response.data || [];
  } catch (e) {
    error.value = e.response?.data || 'Failed to load users';
    console.error('Failed to load users:', e);
  } finally {
    loading.value = false;
  }
};

const openCreateModal = () => {
  isEditMode.value = false;
  formData.value = defaultFormData();
  showModal.value = true;
};

const editUser = (user) => {
  isEditMode.value = true;
  formData.value = { ...user, password: '' };
  showModal.value = true;
};

const saveUser = async () => {
  if (isEditMode.value && !canEditUsers.value) {
    toast.add({ severity: 'warn', summary: 'Forbidden', detail: 'Missing permission user:edit', life: 4000 });
    return;
  }

  if (!isEditMode.value && !canCreateUsers.value) {
    toast.add({ severity: 'warn', summary: 'Forbidden', detail: 'Missing permission user:create', life: 4000 });
    return;
  }

  if (!formData.value.username || !formData.value.full_name || !formData.value.email) {
    toast.add({ severity: 'warn', summary: 'Validation', detail: 'Username, full name, and email are required', life: 4000 });
    return;
  }

  if (!isEditMode.value && !formData.value.password) {
    toast.add({ severity: 'warn', summary: 'Validation', detail: 'Password is required', life: 4000 });
    return;
  }

  saving.value = true;
  try {
    if (isEditMode.value) {
      const updateData = { ...formData.value };
      if (!updateData.password) {
        delete updateData.password;
      }
      await axios.put(`/api/users/${formData.value.id}`, updateData);
      toast.add({ severity: 'success', summary: 'Success', detail: 'User updated', life: 3000 });
    } else {
      await axios.post('/api/users', formData.value);
      toast.add({ severity: 'success', summary: 'Success', detail: 'User created', life: 3000 });
    }
    closeModal();
    await loadUsers();
  } catch (e) {
    const msg = typeof e.response?.data === 'string' ? e.response.data : 'Failed to save user';
    toast.add({ severity: 'error', summary: 'Error', detail: msg, life: 5000 });
  } finally {
    saving.value = false;
  }
};

const toggleUserEnabled = async (user) => {
  if (!canEditUsers.value) {
    toast.add({ severity: 'warn', summary: 'Forbidden', detail: 'Missing permission user:edit', life: 4000 });
    return;
  }

  try {
    await axios.put(`/api/users/${user.id}`, {
      ...user,
      enabled: !user.enabled,
      password: ''
    });
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: `User ${!user.enabled ? 'activated' : 'deactivated'}`,
      life: 3000
    });
    await loadUsers();
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to toggle user status', life: 5000 });
  }
};

const confirmDeleteUser = (user) => {
  if (!canDeleteUsers.value) {
    toast.add({ severity: 'warn', summary: 'Forbidden', detail: 'Missing permission user:delete', life: 4000 });
    return;
  }

  confirm.require({
    message: `Are you sure you want to delete user "${user.username}" (${user.full_name})?`,
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptLabel: 'Delete',
    rejectLabel: 'Cancel',
    accept: async () => {
      try {
        await axios.delete(`/api/users/${user.id}`);
        toast.add({ severity: 'success', summary: 'Success', detail: 'User deleted', life: 3000 });
        await loadUsers();
      } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete user', life: 5000 });
      }
    }
  });
};

const closeModal = () => {
  showModal.value = false;
  formData.value = defaultFormData();
};

const getRoleSeverity = (role) => {
  const severities = {
    admin: 'danger',
    techniker: 'info',
    benutzer: 'success',
    auditor: 'warn'
  }
  return severities[role] || 'secondary'
}

onMounted(() => {
  loadUsers();
});
</script>

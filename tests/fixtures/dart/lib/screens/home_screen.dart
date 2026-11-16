import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../backup/restore.dart';
import '../state/account_model.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  Future<void> _logout(BuildContext context) async {
    try {
      await context.read<AccountModel>().logout();
      if (context.mounted) context.go('/login');
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('Chưa đăng xuất được, thử lại')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final account = context.watch<AccountModel>();
    return Scaffold(
      appBar: AppBar(
        title: Text(account.profile?['name'] ?? 'Giao dịch'),
        actions: [
          IconButton(
            icon: const Icon(Icons.restore),
            onPressed: () => BackupRestore.importFromFile(),
          ),
          IconButton(icon: const Icon(Icons.logout), onPressed: () => _logout(context)),
        ],
      ),
      body: ListView(
        children: [
          for (final tx in account.transactions)
            ListTile(
              title: Text('${tx['amount']} đ'),
              subtitle: Text('${tx['content']}'),
              onTap: () => context.go('/orders/${Uri.encodeComponent('${tx['id']}')}'),
            ),
        ],
      ),
    );
  }
}

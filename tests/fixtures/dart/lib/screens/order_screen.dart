import 'package:flutter/material.dart';

import '../api/client.dart';

class OrderScreen extends StatelessWidget {
  const OrderScreen({super.key, required this.id});

  final String id;

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Chi tiết giao dịch')),
        body: FutureBuilder<dynamic>(
          future: ApiClient.instance.get('/transactions/${Uri.encodeComponent(id)}'),
          builder: (context, snapshot) {
            if (!snapshot.hasData) return const Center(child: CircularProgressIndicator());
            final tx = snapshot.data as Map<String, dynamic>;
            return ListView(
              children: [
                ListTile(title: const Text('Số tiền'), subtitle: Text('${tx['amount']} đ')),
                ListTile(title: const Text('Nội dung'), subtitle: Text('${tx['content']}')),
              ],
            );
          },
        ),
      );
}
